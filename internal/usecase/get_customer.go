package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/repository"
)

type GetCustomer struct {
	customerRepo repository.Customer
}

func NewGetCustomer(customerRepo repository.Customer) *GetCustomer {
	return &GetCustomer{customerRepo: customerRepo}
}

func (uc *GetCustomer) Execute(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	return uc.customerRepo.FindByID(ctx, id)
}
