package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/lib/pq"
)

func Connect(databaseURL string) (*sql.DB, error) {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)

	log.Println("database connection established successfully")
	return db, nil
}

func CreateTables(db *sql.DB) error {
	query := `
	CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

	CREATE TABLE IF NOT EXISTS customers (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		full_name VARCHAR(255) NOT NULL,
		email VARCHAR(255) UNIQUE NOT NULL,
		cpf VARCHAR(11) UNIQUE NOT NULL,
		phone VARCHAR(20),
		birth_date DATE NOT NULL,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW()
	);

	CREATE TABLE IF NOT EXISTS accounts (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		customer_id UUID NOT NULL REFERENCES customers(id) ON DELETE RESTRICT,
		number VARCHAR(50) UNIQUE NOT NULL,
		type VARCHAR(20) NOT NULL,
		balance BIGINT NOT NULL DEFAULT 0,
		daily_limit BIGINT NOT NULL DEFAULT 100000,
		is_active BOOLEAN NOT NULL DEFAULT true,
		blocked_at TIMESTAMP,
		blocked_reason TEXT,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
		CONSTRAINT chk_account_type CHECK (type IN ('checking', 'savings'))
	);

	CREATE INDEX IF NOT EXISTS idx_accounts_customer_id ON accounts(customer_id);
	CREATE INDEX IF NOT EXISTS idx_accounts_number ON accounts(number);

	CREATE TABLE IF NOT EXISTS transactions (
		id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
		account_id UUID NOT NULL REFERENCES accounts(id) ON DELETE RESTRICT,
		transaction_type VARCHAR(20) NOT NULL,
		amount BIGINT NOT NULL,
		balance_before BIGINT NOT NULL,
		balance_after BIGINT NOT NULL,
		description TEXT,
		reference_id UUID,
		metadata JSONB,
		created_at TIMESTAMP NOT NULL DEFAULT NOW(),
		CONSTRAINT chk_transaction_type CHECK (
			transaction_type IN ('deposit', 'withdrawal', 'transfer_in', 'transfer_out', 'payment', 'refund')
		)
	);

	CREATE INDEX IF NOT EXISTS idx_transactions_account_id ON transactions(account_id);
	CREATE INDEX IF NOT EXISTS idx_transactions_created_at ON transactions(created_at DESC);
	`

	_, err := db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to create tables: %w", err)
	}

	log.Println("tables verified/created successfully")
	return nil
}
