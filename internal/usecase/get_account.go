package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/repository"
)

type GetAccount struct {
	accountRepo repository.Account
}

func NewGetAccount(accountRepo repository.Account) *GetAccount {
	return &GetAccount{accountRepo: accountRepo}
}

func (uc *GetAccount) Execute(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	return uc.accountRepo.FindByID(ctx, id)
}
