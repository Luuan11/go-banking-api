package config

import (
	"os"
	"time"
)

type Config struct {
	DatabaseURL     string
	ServerPort      string
	ShutdownTimeout time.Duration
	LogLevel        string
	LogFormat       string
}

func Load() *Config {
	return &Config{
		DatabaseURL:     getEnv("DATABASE_URL", "postgres://bankuser:bankpass@localhost:5432/bankdb?sslmode=disable"),
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		ShutdownTimeout: getDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		LogLevel:        getEnv("LOG_LEVEL", "info"),
		LogFormat:       getEnv("LOG_FORMAT", "json"),
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(key)); err == nil {
		return d
	}
	return defaultValue
}
