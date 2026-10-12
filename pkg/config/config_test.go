package config

import (
	"testing"
	"time"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	t.Setenv("SERVER_PORT", "")
	t.Setenv("SHUTDOWN_TIMEOUT", "")
	t.Setenv("LOG_LEVEL", "")
	t.Setenv("LOG_FORMAT", "")

	cfg := Load()
	if cfg.ServerPort != "8080" || cfg.ShutdownTimeout != 10*time.Second || cfg.DatabaseURL == "" ||
		cfg.LogLevel != "info" || cfg.LogFormat != "json" {
		t.Errorf("unexpected defaults: %+v", cfg)
	}
}

func TestLoadFromEnv(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("SERVER_PORT", "9090")
	t.Setenv("SHUTDOWN_TIMEOUT", "3s")

	cfg := Load()
	if cfg.DatabaseURL != "postgres://example" || cfg.ServerPort != "9090" || cfg.ShutdownTimeout != 3*time.Second {
		t.Errorf("unexpected config: %+v", cfg)
	}
}

func TestInvalidDurationFallsBack(t *testing.T) {
	t.Setenv("SHUTDOWN_TIMEOUT", "soon")
	if got := Load().ShutdownTimeout; got != 10*time.Second {
		t.Errorf("ShutdownTimeout = %v, want 10s", got)
	}
}
