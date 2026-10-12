package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	httpHandler "github.com/luuan11/bank-system/internal/handler/http"
	"github.com/luuan11/bank-system/internal/metrics"
	"github.com/luuan11/bank-system/internal/middleware"
	"github.com/luuan11/bank-system/internal/repository/postgres"
	"github.com/luuan11/bank-system/internal/usecase"
	"github.com/luuan11/bank-system/pkg/config"
	"github.com/luuan11/bank-system/pkg/database"
	"github.com/luuan11/bank-system/pkg/logger"
)

// version is set at build time with -ldflags "-X main.version=...".
var version = "dev"

func main() {
	if err := run(); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	log, err := logger.New(os.Stdout, cfg.LogLevel, cfg.LogFormat, "service", "bank-api", "version", version)
	if err != nil {
		return err
	}
	slog.SetDefault(log)
	slog.Info("starting", "go_version", runtime.Version(), "port", cfg.ServerPort, "log_level", cfg.LogLevel)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()
	slog.Info("database connection established")

	if err := database.Migrate(ctx, db); err != nil {
		return err
	}

	m := metrics.New(version)
	m.RegisterDB(db)

	customerRepo := postgres.NewCustomer(db)
	accountRepo := postgres.NewAccount(db)
	transactionRepo := postgres.NewTransaction(db)
	uow := postgres.NewUnitOfWork(db)

	router := httpHandler.NewRouter(
		httpHandler.NewHealth(db),
		httpHandler.NewCustomer(
			usecase.NewCreateCustomer(customerRepo),
			usecase.NewGetCustomer(customerRepo),
		),
		httpHandler.NewAccount(
			usecase.NewCreateAccount(accountRepo, customerRepo),
			usecase.NewGetAccount(accountRepo),
			usecase.NewListTransactions(accountRepo, transactionRepo),
		),
		httpHandler.NewOperation(
			usecase.NewDeposit(uow),
			usecase.NewWithdraw(uow),
			usecase.NewTransfer(uow),
			m,
		),
		httpHandler.RegisterFunc(func(mux *http.ServeMux) {
			mux.Handle("GET /metrics", m.Handler())
		}),
	)

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           middleware.RequestID(middleware.Logger(middleware.Metrics(m)(middleware.Recovery(middleware.CORS(router))))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server listening", "addr", server.Addr)
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	case <-ctx.Done():
	}

	slog.Info("shutting down", "timeout", cfg.ShutdownTimeout.String())
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
