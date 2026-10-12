package database

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
)

type Options struct {
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	// ConnectTimeout bounds the retries while the database comes up.
	ConnectTimeout time.Duration
}

func (o Options) withDefaults() Options {
	if o.MaxOpenConns <= 0 {
		o.MaxOpenConns = 25
	}
	if o.MaxIdleConns <= 0 {
		o.MaxIdleConns = 10
	}
	if o.ConnMaxLifetime <= 0 {
		o.ConnMaxLifetime = 30 * time.Minute
	}
	if o.ConnMaxIdleTime <= 0 {
		o.ConnMaxIdleTime = 5 * time.Minute
	}
	if o.ConnectTimeout <= 0 {
		o.ConnectTimeout = 30 * time.Second
	}
	return o
}

// Connect retries with exponential backoff until the database answers.
func Connect(ctx context.Context, databaseURL string, opts Options) (*sql.DB, error) {
	opts = opts.withDefaults()

	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	db.SetMaxOpenConns(opts.MaxOpenConns)
	db.SetMaxIdleConns(opts.MaxIdleConns)
	db.SetConnMaxLifetime(opts.ConnMaxLifetime)
	db.SetConnMaxIdleTime(opts.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(ctx, opts.ConnectTimeout)
	defer cancel()

	backoff := 500 * time.Millisecond
	for attempt := 1; ; attempt++ {
		err := db.PingContext(ctx)
		if err == nil {
			return db, nil
		}

		slog.WarnContext(ctx, "database not ready", "attempt", attempt, "retry_in", backoff.String(), "error", err)
		select {
		case <-ctx.Done():
			db.Close()
			return nil, fmt.Errorf("database unreachable after %d attempts: %w", attempt, err)
		case <-time.After(backoff):
		}
		backoff = min(backoff*2, 5*time.Second)
	}
}
