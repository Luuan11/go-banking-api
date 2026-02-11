package main

import (
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

	db, err := database.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.CreateTables(db); err != nil {
		log.Fatalf("failed to create tables: %v", err)
	}

	accountRepo := postgres.NewAccount(db)
	createAccountUC := usecase.NewCreateAccount(accountRepo)
	accountHandler := httpHandler.NewAccount(createAccountUC)

	mux := http.NewServeMux()
	// maybe I created a /health
	mux.HandleFunc("/api/accounts", accountHandler.Create)

	handler := middleware.Recovery(middleware.Logger(middleware.CORS(mux)))

	addr := ":" + cfg.ServerPort
	log.Printf("server starting on port %s", cfg.ServerPort)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("failed to start server: %v", err)
	}
}
