package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	httpHandler "github.com/luuan11/bank-system/internal/handler/http"
	"github.com/luuan11/bank-system/internal/middleware"
	"github.com/luuan11/bank-system/internal/repository/postgres"
	"github.com/luuan11/bank-system/internal/usecase"
	"github.com/luuan11/bank-system/pkg/config"
	"github.com/luuan11/bank-system/pkg/database"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("server stopped with error", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

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
		),
	)

	server := &http.Server{
		Addr:              ":" + cfg.ServerPort,
		Handler:           middleware.Recovery(middleware.Logger(middleware.CORS(router))),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		slog.Info("server starting", "port", cfg.ServerPort)
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

	slog.Info("shutting down server")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	return server.Shutdown(shutdownCtx)
}
