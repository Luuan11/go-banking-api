package main

import (
	"context"
	"log"
	"net/http"

	httpHandler "github.com/luuan11/bank-system/internal/handler/http"
	"github.com/luuan11/bank-system/internal/middleware"
	"github.com/luuan11/bank-system/internal/repository/postgres"
	"github.com/luuan11/bank-system/internal/usecase"
	"github.com/luuan11/bank-system/pkg/config"
	"github.com/luuan11/bank-system/pkg/database"
)

func main() {
	cfg := config.Load()
	log.Println("configuration loaded")

	ctx := context.Background()

	db, err := database.Connect(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	customerRepo := postgres.NewCustomer(db)
	accountRepo := postgres.NewAccount(db)

	customerHandler := httpHandler.NewCustomer(
		usecase.NewCreateCustomer(customerRepo),
		usecase.NewGetCustomer(customerRepo),
	)
	accountHandler := httpHandler.NewAccount(usecase.NewCreateAccount(accountRepo))

	mux := http.NewServeMux()
	customerHandler.Register(mux)
	accountHandler.Register(mux)

	handler := middleware.Recovery(middleware.Logger(middleware.CORS(mux)))

	addr := ":" + cfg.ServerPort
	log.Printf("server starting on port %s", cfg.ServerPort)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
