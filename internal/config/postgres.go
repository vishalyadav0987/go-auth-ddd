package config

import (
	"os"

	"github.com/vishalyadav0987/authentication/internal/infrastructure/persistence/postgres"
)

func LoadPostgresConfig() postgres.Config {
	return postgres.Config{
		Host:     getEnv("DB_HOST", "localhost"),
		Port:     getEnv("DB_PORT", "5432"),
		User:     getEnv("DB_USER", "postgres"),
		Password: getEnv("DB_PASSWORD", ""),
		Database: getEnv("DB_NAME", "order_db"),
		SSLMode:  getEnv("DB_SSLMODE", "disable"),
	}
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}

	return value
}
