package usecase

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/repository"
)

type CreateAccountInput struct {
	CustomerID uuid.UUID          `json:"customer_id"`
	Type       domain.AccountType `json:"type"`
}

type CreateAccountOutput struct {
	Account *domain.Account `json:"account"`
}

type CreateAccount struct {
	accountRepo repository.Account
}

func NewCreateAccount(accountRepo repository.Account) *CreateAccount {
	return &CreateAccount{
		accountRepo: accountRepo,
	}
}

func (uc *CreateAccount) Execute(ctx context.Context, input CreateAccountInput) (*CreateAccountOutput, error) {
	if !input.Type.IsValid() {
		return nil, domain.ErrInvalidAccountType
	}

	if input.CustomerID == uuid.Nil {
		return nil, domain.ErrInvalidCustomerID
	}

	exists, err := uc.accountRepo.ExistsByCustomerID(ctx, input.CustomerID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing account: %w", err)
	}
	if exists {
		return nil, domain.ErrAccountAlreadyExists
	}

	account := domain.NewAccount(input.CustomerID, input.Type)

	accountNumber, err := generateAccountNumber()
	if err != nil {
		return nil, fmt.Errorf("failed to generate account number: %w", err)
	}
	account.Number = accountNumber

	if err := uc.accountRepo.Create(ctx, account); err != nil {
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	return &CreateAccountOutput{Account: account}, nil
}

func generateAccountNumber() (string, error) {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	num := binary.BigEndian.Uint64(b) % 1000000000000
	return fmt.Sprintf("%012d", num), nil
}
