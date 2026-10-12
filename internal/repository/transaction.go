package repository

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
)

type Transaction interface {
	Create(ctx context.Context, transaction *domain.Transaction) error
	// SumDebitsSince returns the total of withdrawals and outgoing transfers
	// of an account since the given instant.
	SumDebitsSince(ctx context.Context, accountID uuid.UUID, since time.Time) (int64, error)
}
