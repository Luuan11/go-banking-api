package postgres

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/luuan11/bank-system/internal/repository"
)

type UnitOfWork struct {
	db *sql.DB
}

func NewUnitOfWork(db *sql.DB) *UnitOfWork {
	return &UnitOfWork{db: db}
}

func (u *UnitOfWork) Do(ctx context.Context, fn func(ctx context.Context, repos repository.Repositories) error) error {
	tx, err := u.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	repos := repository.Repositories{
		Accounts:     NewAccount(tx),
		Transactions: NewTransaction(tx),
	}
	if err := fn(ctx, repos); err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}
	return nil
}
