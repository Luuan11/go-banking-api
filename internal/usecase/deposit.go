package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/repository"
)

// MovementInput is the payload of a deposit or a withdrawal. Amount is in
// cents.
type MovementInput struct {
	Amount      int64  `json:"amount"`
	Description string `json:"description"`
}

type Deposit struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func NewDeposit(uow repository.UnitOfWork) *Deposit {
	return &Deposit{uow: uow, now: domain.Now}
}

func (uc *Deposit) Execute(ctx context.Context, accountID uuid.UUID, input MovementInput) (*domain.Transaction, error) {
	if input.Amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if err := domain.ValidateDescription(input.Description); err != nil {
		return nil, err
	}

	var transaction *domain.Transaction
	err := uc.uow.Do(ctx, func(ctx context.Context, repos repository.Repositories) error {
		account, err := repos.Accounts.FindByIDForUpdate(ctx, accountID)
		if err != nil {
			return err
		}

		transaction, err = account.Credit(input.Amount, domain.TransactionTypeDeposit, input.Description, uc.now().UTC())
		if err != nil {
			return err
		}

		if err := repos.Accounts.UpdateBalance(ctx, account); err != nil {
			return err
		}
		return repos.Transactions.Create(ctx, transaction)
	})
	if err != nil {
		return nil, err
	}

	return transaction, nil
}
