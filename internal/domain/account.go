package domain

import (
	"time"

	"github.com/google/uuid"
)

type AccountType string

const (
	AccountTypeChecking AccountType = "checking"
	AccountTypeSavings  AccountType = "savings"
)

// DefaultDailyLimit is the maximum amount, in cents, that can leave an
// account (withdrawals + outgoing transfers) in a single day.
const DefaultDailyLimit int64 = 100000

type Account struct {
	ID         uuid.UUID   `json:"id"`
	CustomerID uuid.UUID   `json:"customer_id"`
	Number     string      `json:"number"`
	Type       AccountType `json:"type"`
	Balance    int64       `json:"balance"`
	DailyLimit int64       `json:"daily_limit"`
	IsActive   bool        `json:"is_active"`
	BlockedAt  *time.Time  `json:"blocked_at,omitempty"`
	CreatedAt  time.Time   `json:"created_at"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

func NewAccount(customerID uuid.UUID, accountType AccountType) *Account {
	now := Now()
	return &Account{
		ID:         uuid.New(),
		CustomerID: customerID,
		Type:       accountType,
		Balance:    0,
		DailyLimit: DefaultDailyLimit,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func (a AccountType) IsValid() bool {
	return a == AccountTypeChecking || a == AccountTypeSavings
}

func (a *Account) canOperate() error {
	if !a.IsActive || a.BlockedAt != nil {
		return ErrAccountInactive
	}
	return nil
}

// Credit adds amount to the balance and returns the ledger entry for it.
func (a *Account) Credit(amount int64, txType TransactionType, description string, now time.Time) (*Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if err := a.canOperate(); err != nil {
		return nil, err
	}

	return a.apply(amount, txType, description, now), nil
}

// Debit removes amount from the balance. spentToday is the sum of debits
// already made today and is used to enforce the daily limit.
func (a *Account) Debit(amount, spentToday int64, txType TransactionType, description string, now time.Time) (*Transaction, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}
	if err := a.canOperate(); err != nil {
		return nil, err
	}
	if amount > a.Balance {
		return nil, ErrInsufficientBalance
	}
	if spentToday+amount > a.DailyLimit {
		return nil, ErrDailyLimitExceeded
	}

	return a.apply(-amount, txType, description, now), nil
}

func (a *Account) apply(delta int64, txType TransactionType, description string, now time.Time) *Transaction {
	before := a.Balance
	a.Balance += delta
	a.UpdatedAt = now

	amount := delta
	if amount < 0 {
		amount = -amount
	}

	return &Transaction{
		ID:            uuid.New(),
		AccountID:     a.ID,
		Type:          txType,
		Amount:        amount,
		BalanceBefore: before,
		BalanceAfter:  a.Balance,
		Description:   description,
		CreatedAt:     now,
	}
}
