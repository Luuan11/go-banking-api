package postgres

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
)

type Transaction struct {
	db DBTX
}

func NewTransaction(db DBTX) *Transaction {
	return &Transaction{db: db}
}

func (r *Transaction) Create(ctx context.Context, transaction *domain.Transaction) error {
	query := `
		INSERT INTO transactions (id, account_id, transaction_type, amount, balance_before, balance_after, description, reference_id, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, NULLIF($7, ''), $8, $9)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		transaction.ID,
		transaction.AccountID,
		transaction.Type,
		transaction.Amount,
		transaction.BalanceBefore,
		transaction.BalanceAfter,
		transaction.Description,
		transaction.ReferenceID,
		transaction.CreatedAt,
	)
	return err
}

func (r *Transaction) SumDebitsSince(ctx context.Context, accountID uuid.UUID, since time.Time) (int64, error) {
	query := `
		SELECT COALESCE(SUM(amount), 0)
		FROM transactions
		WHERE account_id = $1
		  AND transaction_type IN ('withdrawal', 'transfer_out')
		  AND created_at >= $2
	`

	var total int64
	err := r.db.QueryRowContext(ctx, query, accountID, since).Scan(&total)
	return total, err
}
