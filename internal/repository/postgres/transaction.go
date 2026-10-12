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

func (r *Transaction) ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]domain.Transaction, int, error) {
	var total int
	if err := r.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM transactions WHERE account_id = $1`,
		accountID,
	).Scan(&total); err != nil {
		return nil, 0, err
	}

	query := `
		SELECT id, account_id, transaction_type, amount, balance_before, balance_after,
		       COALESCE(description, ''), reference_id, created_at
		FROM transactions
		WHERE account_id = $1
		ORDER BY created_at DESC, id
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.QueryContext(ctx, query, accountID, limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	transactions := make([]domain.Transaction, 0, limit)
	for rows.Next() {
		var t domain.Transaction
		if err := rows.Scan(
			&t.ID,
			&t.AccountID,
			&t.Type,
			&t.Amount,
			&t.BalanceBefore,
			&t.BalanceAfter,
			&t.Description,
			&t.ReferenceID,
			&t.CreatedAt,
		); err != nil {
			return nil, 0, err
		}
		t.CreatedAt = t.CreatedAt.UTC()
		transactions = append(transactions, t)
	}

	return transactions, total, rows.Err()
}
