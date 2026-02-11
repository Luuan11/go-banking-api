package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
)

type Account struct {
	db *sql.DB
}

func NewAccount(db *sql.DB) *Account {
	return &Account{db: db}
}

func (r *Account) Create(ctx context.Context, account *domain.Account) error {
	query := `
		INSERT INTO accounts (id, customer_id, number, type, balance, daily_limit, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		account.ID,
		account.CustomerID,
		account.Number,
		account.Type,
		account.Balance,
		account.DailyLimit,
		account.IsActive,
		account.CreatedAt,
		account.UpdatedAt,
	)

	return err
}

func (r *Account) FindByID(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	query := `
		SELECT id, customer_id, number, type, balance, daily_limit, is_active, blocked_at, created_at, updated_at 
		FROM accounts WHERE id = $1
	`
	return r.scanAccount(r.db.QueryRowContext(ctx, query, id))
}

func (r *Account) FindByNumber(ctx context.Context, number string) (*domain.Account, error) {
	query := `
		SELECT id, customer_id, number, type, balance, daily_limit, is_active, blocked_at, created_at, updated_at 
		FROM accounts WHERE number = $1
	`
	return r.scanAccount(r.db.QueryRowContext(ctx, query, number))
}

func (r *Account) ExistsByCustomerID(ctx context.Context, customerID uuid.UUID) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM accounts WHERE customer_id = $1)`, customerID).Scan(&exists)
	return exists, err
}

func (r *Account) scanAccount(row *sql.Row) (*domain.Account, error) {
	account := &domain.Account{}
	err := row.Scan(
		&account.ID,
		&account.CustomerID,
		&account.Number,
		&account.Type,
		&account.Balance,
		&account.DailyLimit,
		&account.IsActive,
		&account.BlockedAt,
		&account.CreatedAt,
		&account.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAccountNotFound
		}
		return nil, err
	}
	return account, nil
}
