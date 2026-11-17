package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	authapp "github.com/vishalyadav0987/authentication/internal/application/auth"
	notificationApp "github.com/vishalyadav0987/authentication/internal/application/notification"
	"github.com/vishalyadav0987/authentication/internal/application/order"
	"github.com/vishalyadav0987/authentication/internal/application/payment"
	"github.com/vishalyadav0987/authentication/internal/config"
	domainNotification "github.com/vishalyadav0987/authentication/internal/domain/notification"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/email/brevo"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/hash"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/id"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/notify/email"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/otp"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/payment/cashfree"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/persistence/postgres"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/queue"
	notificationQueue "github.com/vishalyadav0987/authentication/internal/infrastructure/queue/notification"
	orderQueue "github.com/vishalyadav0987/authentication/internal/infrastructure/queue/order"
	postgresRate "github.com/vishalyadav0987/authentication/internal/infrastructure/rate_limiter/postgres"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/token"
	outboxWorker "github.com/vishalyadav0987/authentication/internal/infrastructure/worker/outbox"
	httpApp "github.com/vishalyadav0987/authentication/internal/interfaces/http"
	"github.com/vishalyadav0987/authentication/internal/interfaces/http/handler"
	"github.com/vishalyadav0987/authentication/internal/interfaces/http/middleware"
	"github.com/vishalyadav0987/authentication/pkg/notiflog"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 0. load env data
	cfg := config.MustLoad()

	// Notification-flow structured logging (see pkg/notiflog).
	closeNotificationLog, err := notiflog.Init()
	if err != nil {
		log.Fatalf("notification logger: %v", err)
	}
	defer closeNotificationLog()

	// 0.5 PostgreSQL db connection
	db, err := postgres.NewConnection(config.LoadPostgresConfig())
	if err != nil {
		log.Fatalf("PostgreSQL connection failed: %v", err)
	}
	defer db.Close()

	// 0.8 Run migrations
	postgres.RunMigration(db)

	// 0.9 domain layer
	userRepo := postgres.NewUserRepository(db)
	hasher := hash.NewBcryptHasher(bcrypt.DefaultCost)
	idGen := id.NewUUIDGenerator()
	jwtManager := token.NewJWTManager(cfg.JWTSecret)
	mpinRateLimiter := postgresRate.NewMPINRateLimiter(db)
	otpRepo := postgres.NewOtpRepository(db)
	otpService := otp.NewOTPService(otpRepo, *hash.NewBcryptHasher(bcrypt.DefaultCost))
	emailService := email.NewSMTPEmailService()

	registerUC := authapp.NewRegisterUsecase(userRepo, hasher, idGen)
	loginUC := authapp.NewLoginPasswordUsecase(userRepo, hasher, jwtManager)
	loginWithMpinUC := authapp.NewLoginMPINUsecase(userRepo, hasher, jwtManager, mpinRateLimiter)
	requestOtpUC := authapp.NewRequestOtpUsecase(otpService, jwtManager, emailService)
	verifyOtpUC := authapp.NewVerifyOtpUsecase(otpService, jwtManager, userRepo, hasher)
	createPasswordUC := authapp.NewResetPasswordUsecase(userRepo, hasher, jwtManager)

	authHandler := handler.NewAuthHandler(
		registerUC,
		loginUC,
		loginWithMpinUC,
		verifyOtpUC,
		requestOtpUC,
		createPasswordUC,
		userRepo,
	)

	// Redis and Asynq
	redisConfig := config.LoadRedisConfig()

	queueClient := queue.NewQueueClient(redisConfig.Addr)
	defer queueClient.Close()

	// Order module dependencies
	orderRepo := postgres.NewOrderRepository(db)
	orderIDGenerator := id.NewOrderIDGenerator()

	createOrderUC := order.NewCreateOrderUseCase(
		orderRepo,
		orderIDGenerator,
	)

	paymentRepo := postgres.NewPaymentRepository(db)
	paymentIDGenerator := id.NewPaymentIDGenerator()

	cashfreeClient := cashfree.NewClient(cfg.CashfreeConfig)

	createPaymentUC := payment.NewCreatePaymentUseCase(
		orderRepo,
		paymentRepo,
		cashfreeClient,
		paymentIDGenerator,
	)

	// Extract authenticated user ID from JWT
	getUserID := func(c *gin.Context) (string, error) {
		authHeader := strings.TrimSpace(c.GetHeader("Authorization"))
		if authHeader == "" {
			return "", errors.New("authorization header is required")
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			return "", errors.New("invalid authorization header format, expected 'Bearer <token>'")
		}

		tokenString := strings.TrimSpace(parts[1])
		if tokenString == "" {
			return "", errors.New("bearer token is empty")
		}

		claims, err := jwtManager.VerifyToken(tokenString)
		if err != nil {
			return "", fmt.Errorf("invalid or expired token: %w", err)
		}

		// Reject OTP / reset-password tokens: only access tokens may
		// call order and payment endpoints.
		if claims.TokenType != authapp.TokenTypeAccessToken {
			return "", errors.New("access token required")
		}

		if strings.TrimSpace(claims.UserID) == "" {
			return "", errors.New("user ID missing from token")
		}

		return claims.UserID, nil
	}

	orderHandler := handler.NewOrderHandler(
		createOrderUC,
		getUserID,
	)

	paymentHandler := handler.NewPaymentHandler(
		createPaymentUC,
		getUserID,
	)

	handleWebhookUseCase := payment.NewHandleWebhookUseCase(
		paymentRepo,
	)

	webhookHandler := handler.NewPaymentWebhookHandler(
		cashfreeClient,
		handleWebhookUseCase,
	)

	deadLetterHandler := handler.NewDeadLetterHandler(
		notificationApp.NewGetDeadLetterDeliveriesUseCase(
			postgres.NewNotificationRepository(db),
		),
	)

	deadLetterDetailHandler := handler.NewDeadLetterDetailHandler(
		notificationApp.NewGetDeadLetterDetailUseCase(
			postgres.NewNotificationRepository(db),
			postgres.NewDeliveryAttemptRepository(db),
		),
	)

	retryDeadLetterHandler := handler.NewRetryDeadLetterHandler(
		notificationApp.NewRetryDeadLetterDeliveryUseCase(
			postgres.NewNotificationRepository(db),
		),
	)

	// User notification APIs: inbox, preferences, push devices.
	notificationRepo := postgres.NewNotificationRepository(db)
	deviceRepo := postgres.NewDeviceRepository(db)
	preferenceRepo := postgres.NewPreferenceRepository(db)

	userNotificationHandler := handler.NewUserNotificationHandler(
		notificationApp.NewUserInboxUseCase(postgres.NewInboxRepository(db)),
		notificationApp.NewUserPreferencesUseCase(preferenceRepo),
		notificationApp.NewUserDeviceUseCase(deviceRepo),
	)

	// Admin: create notifications (send flow).
	sendNotificationHandler := handler.NewSendNotificationHandler(
		notificationApp.NewSendNotificationUseCase(
			notificationRepo,
			notificationApp.NewResolveRecipientsUseCase(preferenceRepo),
			userRepo,
			deviceRepo,
		),
	)

	// Register HTTP routes
	router := httpApp.SetUpRouter(
		authHandler,
		orderHandler,
		paymentHandler,
		webhookHandler,
		deadLetterHandler,
		deadLetterDetailHandler,
		retryDeadLetterHandler,
		userNotificationHandler,
		sendNotificationHandler,
		middleware.RequireAccessToken(jwtManager),
		middleware.RequireAdmin(userRepo),
	)

	// 1. creating Router
	// router := gin.New()

	// 2. Basic Middleware
	router.Use(gin.Logger(), gin.Recovery())

	// 3. Testing Routes for Health Check
	router.GET("/test", func(ctx *gin.Context) {
		ctx.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Testing Routes for Health Check",
		})
	})

	// Order transactional outbox (outbox_events)
	orderOutboxRepo := postgres.NewOrderOutboxRepository(db)
	orderEventPublisher := orderQueue.NewOrderEventPublisher(queueClient)

	orderOutboxProcessor := order.NewOutboxProcessor(
		orderOutboxRepo,
		orderEventPublisher,
	)

	orderOutboxWorker := order.NewOutboxWorker(orderOutboxProcessor)

	// Notification transactional outbox (notification_outbox_events)
	notificationOutboxRepo := postgres.NewNotificationOutboxRepository(db)

	// Register every notification outbox handler once, before the
	// worker starts. Order/payment events use the order outbox above.
	notificationOutboxDispatcher := notificationApp.NewOutboxEventDispatcher()

	deliveryPublisher := notificationQueue.NewPublisher(queueClient)

	if err := notificationOutboxDispatcher.Register(
		domainNotification.OutboxEventDeliveryRetryRequested,
		notificationApp.NewDeliveryRetryEventHandler(
			notificationApp.NewProcessDeliveryRetryEventUseCase(
				notificationRepo,
				deliveryPublisher,
			),
		),
	); err != nil {
		log.Fatalf("register notification outbox handler: %v", err)
	}

	if err := notificationOutboxDispatcher.Register(
		domainNotification.OutboxEventNotificationCreated,
		notificationApp.NewNotificationCreatedEventHandler(
			notificationRepo,
			deliveryPublisher,
		),
	); err != nil {
		log.Fatalf("register notification outbox handler: %v", err)
	}

	notificationOutboxWorker, err := outboxWorker.NewWorker(
		notificationOutboxRepo,
		notificationApp.NewProcessOutboxEventUseCase(
			notificationOutboxRepo,
			notificationOutboxDispatcher,
			notificationApp.NewOutboxHeartbeat(notificationOutboxRepo),
		),
		50,
		2*time.Second,
	)
	if err != nil {
		log.Fatalf("create notification outbox worker: %v", err)
	}

	workerCtx, cancelWorkers := context.WithCancel(
		context.Background(),
	)
	defer cancelWorkers()

	// Start outbox polling workers
	go orderOutboxWorker.Start(workerCtx)
	go func() {
		if err := notificationOutboxWorker.Run(workerCtx); err != nil {
			log.Printf("Notification outbox worker exited: %v", err)
		}
	}()

	// Notification delivery processing (Asynq consumer side).
	//
	// Assign the email provider only on success: a nil
	// *SMTPEmailProvider stored in the interface would be non-nil and
	// bypass the use case's "provider not configured" check.
	var emailProvider domainNotification.EmailProvider
	if smtpProvider, err := brevo.NewSMTPEmailProvider(); err != nil {
		log.Printf("Email provider disabled: %v", err)
	} else {
		emailProvider = smtpProvider
	}

	// TODO: no PushProvider (FCM) implementation exists yet. Push
	// deliveries fail with "push provider is not configured" and follow
	// the normal retry / DLQ path until one is wired here.
	processDeliveryUC := notificationApp.NewProcessDeliveryUseCase(
		postgres.NewNotificationRepository(db),
		emailProvider,
		nil, // push provider
		postgres.NewDeliveryAttemptRepository(db),
		deviceRepo,
	)

	// One shared Asynq server and mux for every module.
	asynqMux := asynq.NewServeMux()

	if err := orderQueue.RegisterHandlers(asynqMux); err != nil {
		log.Fatalf("register order queue handlers: %v", err)
	}

	if err := notificationQueue.RegisterHandlers(asynqMux, processDeliveryUC); err != nil {
		log.Fatalf("register notification queue handlers: %v", err)
	}

	// Start Asynq worker
	go func() {
		log.Println("Starting Asynq worker...")

		if err := queue.NewWorkerServer(redisConfig.Addr).Run(asynqMux); err != nil {
			log.Printf("Asynq worker stopped: %v", err)
		}
	}()

	// 4. Create server
	server := &http.Server{
		Addr:    ":" + cfg.AppPort,
		Handler: router,
	}

	// 5. Run server in goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	// 6. Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	log.Println("Server exiting")
}
