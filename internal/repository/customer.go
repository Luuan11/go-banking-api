package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
)

type Customer interface {
	Create(ctx context.Context, customer *domain.Customer) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error)
}
