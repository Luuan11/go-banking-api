package domain

import (
	"time"

	"github.com/google/uuid"
)

const MaxDescriptionLength = 255

type TransactionType string

const (
	TransactionTypeDeposit     TransactionType = "deposit"
	TransactionTypeWithdrawal  TransactionType = "withdrawal"
	TransactionTypeTransferIn  TransactionType = "transfer_in"
	TransactionTypeTransferOut TransactionType = "transfer_out"
)

// Transaction is an immutable ledger entry. Amount is always positive; the
// direction is given by Type and by the balance_before/balance_after pair.
type Transaction struct {
	ID            uuid.UUID       `json:"id"`
	AccountID     uuid.UUID       `json:"account_id"`
	Type          TransactionType `json:"type"`
	Amount        int64           `json:"amount"`
	BalanceBefore int64           `json:"balance_before"`
	BalanceAfter  int64           `json:"balance_after"`
	Description   string          `json:"description,omitempty"`
	ReferenceID   *uuid.UUID      `json:"reference_id,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

func ValidateDescription(description string) error {
	if len([]rune(description)) > MaxDescriptionLength {
		return ErrInvalidDescription
	}
	return nil
}
