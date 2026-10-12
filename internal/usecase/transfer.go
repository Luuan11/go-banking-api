package usecase

import (
	"bytes"
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/repository"
)

type TransferInput struct {
	FromAccountID uuid.UUID `json:"from_account_id"`
	ToAccountID   uuid.UUID `json:"to_account_id"`
	Amount        int64     `json:"amount"`
	Description   string    `json:"description"`
}

type TransferOutput struct {
	ID     uuid.UUID           `json:"id"`
	Debit  *domain.Transaction `json:"debit"`
	Credit *domain.Transaction `json:"credit"`
}

type Transfer struct {
	uow repository.UnitOfWork
	now func() time.Time
}

func NewTransfer(uow repository.UnitOfWork) *Transfer {
	return &Transfer{uow: uow, now: domain.Now}
}

func (uc *Transfer) Execute(ctx context.Context, input TransferInput) (*TransferOutput, error) {
	if input.FromAccountID == uuid.Nil || input.ToAccountID == uuid.Nil {
		return nil, domain.ErrInvalidAccountID
	}
	if input.FromAccountID == input.ToAccountID {
		return nil, domain.ErrSameAccountTransfer
	}
	if input.Amount <= 0 {
		return nil, domain.ErrInvalidAmount
	}
	if err := domain.ValidateDescription(input.Description); err != nil {
		return nil, err
	}

	output := &TransferOutput{ID: uuid.New()}
	err := uc.uow.Do(ctx, func(ctx context.Context, repos repository.Repositories) error {
		now := uc.now().UTC()

		from, to, err := lockPair(ctx, repos.Accounts, input.FromAccountID, input.ToAccountID)
		if err != nil {
			return err
		}

		spentToday, err := repos.Transactions.SumDebitsSince(ctx, from.ID, startOfDay(now))
		if err != nil {
			return err
		}

		debit, err := from.Debit(input.Amount, spentToday, domain.TransactionTypeTransferOut, input.Description, now)
		if err != nil {
			return err
		}
		credit, err := to.Credit(input.Amount, domain.TransactionTypeTransferIn, input.Description, now)
		if err != nil {
			return err
		}

		debit.ReferenceID = &output.ID
		credit.ReferenceID = &output.ID

		for _, account := range []*domain.Account{from, to} {
			if err := repos.Accounts.UpdateBalance(ctx, account); err != nil {
				return err
			}
		}
		for _, transaction := range []*domain.Transaction{debit, credit} {
			if err := repos.Transactions.Create(ctx, transaction); err != nil {
				return err
			}
		}

		output.Debit = debit
		output.Credit = credit
		return nil
	})
	if err != nil {
		return nil, err
	}

	return output, nil
}

// lockPair locks in id order so opposite transfers cannot deadlock.
func lockPair(ctx context.Context, accounts repository.Account, fromID, toID uuid.UUID) (from, to *domain.Account, err error) {
	firstID, secondID := fromID, toID
	if bytes.Compare(firstID[:], secondID[:]) > 0 {
		firstID, secondID = secondID, firstID
	}

	first, err := accounts.FindByIDForUpdate(ctx, firstID)
	if err != nil {
		return nil, nil, err
	}
	second, err := accounts.FindByIDForUpdate(ctx, secondID)
	if err != nil {
		return nil, nil, err
	}

	if first.ID == fromID {
		return first, second, nil
	}
	return second, first, nil
}
