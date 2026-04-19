package config

import (
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	AppPort         string
	DBPath          string
	JWTSecret       string
	AccessTokenTTL  time.Duration
	RefreshTokenTTL time.Duration
}

func MustLoad() *Config {
	err := godotenv.Load()

	if err != nil {
		log.Println("No .env file found")
	}

	// Converstion of string to duration
	accessTTL, _ := time.ParseDuration(os.Getenv("ACCESS_TOKEN_TTL"))
	refreshTTL, _ := time.ParseDuration(os.Getenv("REFRESH_TOKEN_TTL"))

	return &Config{
		AppPort:         os.Getenv("APP_PORT"),
		DBPath:          os.Getenv("DB_PATH"),
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  accessTTL,
		RefreshTokenTTL: refreshTTL,
	}
}
