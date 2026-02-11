package repository

import (
	"context"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
)

type Account interface {
	Create(ctx context.Context, account *domain.Account) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Account, error)
	FindByNumber(ctx context.Context, number string) (*domain.Account, error)
	ExistsByCustomerID(ctx context.Context, customerID uuid.UUID) (bool, error)
}
