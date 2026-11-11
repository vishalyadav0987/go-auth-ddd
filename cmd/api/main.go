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
	authapp "github.com/vishalyadav0987/authentication/internal/application/auth"
	"github.com/vishalyadav0987/authentication/internal/application/order"
	"github.com/vishalyadav0987/authentication/internal/config"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/hash"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/id"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/notify/email"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/otp"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/persistence/postgres"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/queue"
	postgresRate "github.com/vishalyadav0987/authentication/internal/infrastructure/rate_limiter/postgres"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/token"
	httpApp "github.com/vishalyadav0987/authentication/internal/interfaces/http"
	"github.com/vishalyadav0987/authentication/internal/interfaces/http/handler"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 0. load env data
	cfg := config.MustLoad()

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

		if strings.TrimSpace(claims.UserID) == "" {
			return "", errors.New("user ID missing from token")
		}

		return claims.UserID, nil
	}

	orderHandler := handler.NewOrderHandler(
		createOrderUC,
		getUserID,
	)

	// Register HTTP routes
	router := httpApp.SetUpRouter(
		authHandler,
		orderHandler,
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

	// Transactional Outbox dependencies
	outboxRepo := postgres.NewOutboxRepository(db)
	eventPublisher := queue.NewOrderEventPublisher(queueClient)

	outboxProcessor := order.NewOutboxProcessor(
		outboxRepo,
		eventPublisher,
	)

	outboxWorker := order.NewOutboxWorker(outboxProcessor)

	workerCtx, cancelWorkers := context.WithCancel(
		context.Background(),
	)
	defer cancelWorkers()

	// Start Outbox polling worker
	go outboxWorker.Start(workerCtx)

	// Start Asynq worker
	go func() {
		log.Println("Starting Asynq worker...")

		if err := queue.StartWorker(redisConfig.Addr); err != nil {
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
