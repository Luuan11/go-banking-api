package config

import (
	"os"
	"strconv"
	"time"

	"github.com/luuan11/bank-system/pkg/database"
)

type Config struct {
	DatabaseURL     string
	Database        database.Options
	ServerPort      string
	ShutdownTimeout time.Duration
	// DrainDelay keeps serving after readiness fails, so load balancers notice.
	DrainDelay time.Duration
	LogLevel   string
	LogFormat  string
}

func Load() *Config {
	return &Config{
		DatabaseURL: getEnv("DATABASE_URL", "postgres://bankuser:bankpass@localhost:5432/bankdb?sslmode=disable"),
		Database: database.Options{
			MaxOpenConns:    getInt("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getInt("DB_MAX_IDLE_CONNS", 10),
			ConnMaxLifetime: getDuration("DB_CONN_MAX_LIFETIME", 30*time.Minute),
			ConnMaxIdleTime: getDuration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
			ConnectTimeout:  getDuration("DB_CONNECT_TIMEOUT", 30*time.Second),
		},
		ServerPort:      getEnv("SERVER_PORT", "8080"),
		ShutdownTimeout: getDuration("SHUTDOWN_TIMEOUT", 10*time.Second),
		DrainDelay:      getDuration("DRAIN_DELAY", 0),
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

func getInt(key string, defaultValue int) int {
	if n, err := strconv.Atoi(os.Getenv(key)); err == nil {
		return n
	}
	return defaultValue
}

func getDuration(key string, defaultValue time.Duration) time.Duration {
	if d, err := time.ParseDuration(os.Getenv(key)); err == nil {
		return d
	}
	return defaultValue
}
