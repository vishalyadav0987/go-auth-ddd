package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/vishalyadav0987/authentication/internal/infrastructure/payment/cashfree"
)

type Config struct {
	AppPort         string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
	CashfreeConfig  cashfree.Config
}

func MustLoad() *Config {
	err := godotenv.Load()

	if err != nil {
		log.Println("No .env file found")
	}

	// Converstion of string to duration
	accessTTL, _ := time.ParseDuration(os.Getenv("ACCESS_TOKEN_TTL"))
	refreshTTL, _ := time.ParseDuration(os.Getenv("REFRESH_TOKEN_TTL"))

	baseURL := "https://sandbox.cashfree.com/pg"
	if os.Getenv("CASHFREE_ENV") == "production" {
		baseURL = "https://api.cashfree.com/pg"
	}

	apiVersion := os.Getenv("CASHFREE_API_VERSION")
	if apiVersion == "" {
		apiVersion = "2023-08-01"
	}

	return &Config{
		AppPort:         os.Getenv("APP_PORT"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
		CashfreeConfig: cashfree.Config{
			ClientID:     os.Getenv("CASHFREE_APP_ID"),
			ClientSecret: os.Getenv("CASHFREE_SECRET_KEY"),
			BaseURL:      baseURL,
			APIVersion:   apiVersion,
		},
	}
}
