package usecase

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/repository"
)

type Withdraw struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func NewWithdraw(uow repository.UnitOfWork) *Withdraw {
	return &Withdraw{uow: uow, now: time.Now}
}

func (uc *Withdraw) Execute(ctx context.Context, accountID uuid.UUID, input MovementInput) (*domain.Transaction, error) {
	if input.Amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if err := domain.ValidateDescription(input.Description); err != nil {
		return nil, err
	}

	var transaction *domain.Transaction
	err := uc.uow.Do(ctx, func(ctx context.Context, repos repository.Repositories) error {
		now := uc.now().UTC()

		// The row lock serializes every debit on this account, which also
		// makes the daily limit sum below consistent.
		account, err := repos.Accounts.FindByIDForUpdate(ctx, accountID)
		if err != nil {
			return err
		}

		spentToday, err := repos.Transactions.SumDebitsSince(ctx, account.ID, startOfDay(now))
		if err != nil {
			return err
		}

		transaction, err = account.Debit(input.Amount, spentToday, domain.TransactionTypeWithdrawal, input.Description, now)
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

// startOfDay returns midnight UTC of the day t falls in; the daily limit
// resets at that instant.
func startOfDay(t time.Time) time.Time {
	t = t.UTC()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}
