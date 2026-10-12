package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
)

type Transaction interface {
	Create(ctx context.Context, transaction *domain.Transaction) error
	// SumDebitsSince sums withdrawals and outgoing transfers.
	SumDebitsSince(ctx context.Context, accountID uuid.UUID, since time.Time) (int64, error)
	// ListByAccount returns a newest-first page and the total count.
	ListByAccount(ctx context.Context, accountID uuid.UUID, limit, offset int) ([]domain.Transaction, int, error)
}
