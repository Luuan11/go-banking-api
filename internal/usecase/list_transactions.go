package usecase

import (
	"context"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/repository"
)

const (
	DefaultPageSize = 20
	MaxPageSize     = 100
)

type ListTransactionsInput struct {
	AccountID uuid.UUID
	Limit     int
	Offset    int
}

type ListTransactionsOutput struct {
	Items  []domain.Transaction `json:"items"`
	Total  int                  `json:"total"`
	Limit  int                  `json:"limit"`
	Offset int                  `json:"offset"`
}

type ListTransactions struct {
	accountRepo     repository.Account
	transactionRepo repository.Transaction
}

func NewListTransactions(accountRepo repository.Account, transactionRepo repository.Transaction) *ListTransactions {
	return &ListTransactions{
		accountRepo:     accountRepo,
		transactionRepo: transactionRepo,
	}
}

func (uc *ListTransactions) Execute(ctx context.Context, input ListTransactionsInput) (*ListTransactionsOutput, error) {
	if input.Limit <= 0 {
		input.Limit = DefaultPageSize
	}
	if input.Limit > MaxPageSize {
		input.Limit = MaxPageSize
	}
	if input.Offset < 0 {
		input.Offset = 0
	}

	if _, err := uc.accountRepo.FindByID(ctx, input.AccountID); err != nil {
		return nil, err
	}

	items, total, err := uc.transactionRepo.ListByAccount(ctx, input.AccountID, input.Limit, input.Offset)
	if err != nil {
		return nil, err
	}

	return &ListTransactionsOutput{
		Items:  items,
		Total:  total,
		Limit:  input.Limit,
		Offset: input.Offset,
	}, nil
}
