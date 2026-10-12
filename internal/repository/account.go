package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
)

// ErrAccountNumberTaken is returned by Create when the generated account
// number collides with an existing one, so the caller can retry.
var ErrAccountNumberTaken = errors.New("account number already taken")

type Account interface {
	Create(ctx context.Context, account *domain.Account) error
	FindByID(ctx context.Context, id uuid.UUID) (*domain.Account, error)
	// FindByIDForUpdate locks the account row until the surrounding
	// transaction ends. Only meaningful inside a UnitOfWork.
	FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Account, error)
	UpdateBalance(ctx context.Context, account *domain.Account) error
	FindByNumber(ctx context.Context, number string) (*domain.Account, error)
	ExistsByCustomerAndType(ctx context.Context, customerID uuid.UUID, accountType domain.AccountType) (bool, error)
}
