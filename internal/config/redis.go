package config

import (
	"os"
)

type RedisConfig struct {
	Addr string
}

func LoadRedisConfig() RedisConfig {
	addr := os.Getenv("REDIS_ADDR")

	if addr == "" {
		addr = "localhost:6379"
	}

	return RedisConfig{
		Addr: addr,
	}
}
