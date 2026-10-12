package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
)

// ErrAccountNumberTaken signals a number collision; the caller should retry.
var ErrAccountNumberTaken = errors.New("account number already taken")

type Account interface {
	Create(ctx context.Context, account *domain.Account) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Account, error)
	// FindByIDForUpdate locks the row until the UnitOfWork ends.
	FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Account, error)
	UpdateBalance(ctx context.Context, account *domain.Account) error
	FindByNumber(ctx context.Context, number string) (*domain.Account, error)
	ExistsByCustomerAndType(ctx context.Context, customerID uuid.UUID, accountType domain.AccountType) (bool, error)
}
