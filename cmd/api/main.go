package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	authapp "github.com/vishalyadav0987/authentication/internal/application/auth"
	"github.com/vishalyadav0987/authentication/internal/config"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/hash"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/id"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/notify/email"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/otp"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/persistence/sqlite"
	sqliteRate "github.com/vishalyadav0987/authentication/internal/infrastructure/rate_limiter/sqlite"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/token"
	httpApp "github.com/vishalyadav0987/authentication/internal/interfaces/http"
	"github.com/vishalyadav0987/authentication/internal/interfaces/http/handler"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	// 0. load env data
	cfg := config.MustLoad()

	// 0.5 db connection
	db := sqlite.NewConnection(cfg.DBPath)
	defer db.Close()

	// 0.8 migration
	sqlite.RunMigration(db, cfg.DBPath)

	// 0.9 domain layer
	userRepo := sqlite.NewUserRepository(db)
	hasher := hash.NewBcryptHasher(bcrypt.DefaultCost)
	idGen := id.NewUUIDGenerator()
	jwtManager := token.NewJWTManager(cfg.JWTSecret)
	mpinRateLimiter := sqliteRate.NewMPINRateLimiter(db)
	otpRepo := sqlite.NewOtpRepository(db)
	otpService := otp.NewOTPService(otpRepo, *hash.NewBcryptHasher(bcrypt.DefaultCost))
	emailService := email.NewSMTPEmailService()

	registerUC := authapp.NewRegisterUsecase(userRepo, hasher, idGen)
	loginUC := authapp.NewLoginPasswordUsecase(userRepo, hasher, jwtManager)
	loginWithMpinUC := authapp.NewLoginMPINUsecase(userRepo, hasher, jwtManager, mpinRateLimiter)
	requestOtpUC := authapp.NewRequestOtpUsecase(otpService, jwtManager, emailService)
	verifyOtpUC := authapp.NewVerifyOtpUsecase(otpService, jwtManager, userRepo, hasher)
	createPasswordUC := authapp.NewResetPasswordUsecase(userRepo, hasher, jwtManager)

	authHandler := handler.NewAuthHandler(registerUC, loginUC, loginWithMpinUC, verifyOtpUC, requestOtpUC, createPasswordUC, userRepo)

	router := httpApp.SetUpRouter(authHandler)

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
