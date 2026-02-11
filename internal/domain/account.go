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
	now := time.Now()
	return &Account{
		ID:         uuid.New(),
		CustomerID: customerID,
		Type:       accountType,
		Balance:    0,
		DailyLimit: 100000,
		IsActive:   true,
		CreatedAt:  now,
		UpdatedAt:  now,
	}
}

func (a AccountType) IsValid() bool {
	return a == AccountTypeChecking || a == AccountTypeSavings
}
