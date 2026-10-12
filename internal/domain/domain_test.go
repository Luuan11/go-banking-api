package domain

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIsValidCPF(t *testing.T) {
	tests := []struct {
		cpf  string
		want bool
	}{
		{"52998224725", true},
		{"11144477735", true},
		{"52998224726", false},
		{"11111111111", false},
		{"1234567890", false},
		{"5299822472a", false},
		{"", false},
	}

	for _, tt := range tests {
		if got := IsValidCPF(tt.cpf); got != tt.want {
			t.Errorf("IsValidCPF(%q) = %v, want %v", tt.cpf, got, tt.want)
		}
	}
}

func TestNormalizeCPF(t *testing.T) {
	if got := NormalizeCPF("529.982.247-25"); got != "52998224725" {
		t.Errorf("NormalizeCPF = %q, want 52998224725", got)
	}
}

func TestNewCustomer(t *testing.T) {
	now := time.Date(2026, 10, 12, 0, 0, 0, 0, time.UTC)
	adult := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		fullName  string
		email     string
		cpf       string
		birthDate time.Time
		wantErr   error
	}{
		{"valid", "Maria Silva", "Maria@Example.com", "529.982.247-25", adult, nil},
		{"empty name", "  ", "maria@example.com", "52998224725", adult, ErrInvalidCustomerName},
		{"bad email", "Maria", "maria", "52998224725", adult, ErrInvalidEmail},
		{"email with display name", "Maria", "Maria <maria@example.com>", "52998224725", adult, ErrInvalidEmail},
		{"bad cpf", "Maria", "maria@example.com", "12345678900", adult, ErrInvalidCPF},
		{"underage", "Maria", "maria@example.com", "52998224725", now.AddDate(-17, 0, 0), ErrInvalidBirthDate},
		{"exactly 18", "Maria", "maria@example.com", "52998224725", now.AddDate(-18, 0, 0), nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			customer, err := NewCustomer(tt.fullName, tt.email, tt.cpf, "", tt.birthDate, now)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if err == nil {
				if customer.CPF != "52998224725" {
					t.Errorf("cpf not normalized: %q", customer.CPF)
				}
				if customer.Email != "maria@example.com" {
					t.Errorf("email not normalized: %q", customer.Email)
				}
			}
		})
	}
}

func TestAccountCredit(t *testing.T) {
	now := time.Now().UTC()
	account := NewAccount(uuid.New(), AccountTypeChecking)

	tx, err := account.Credit(500, TransactionTypeDeposit, "salary", now)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if account.Balance != 500 {
		t.Errorf("balance = %d, want 500", account.Balance)
	}
	if tx.Amount != 500 || tx.BalanceBefore != 0 || tx.BalanceAfter != 500 || tx.AccountID != account.ID {
		t.Errorf("unexpected transaction: %+v", tx)
	}

	if _, err := account.Credit(0, TransactionTypeDeposit, "", now); !errors.Is(err, ErrInvalidAmount) {
		t.Errorf("zero amount err = %v, want ErrInvalidAmount", err)
	}

	account.IsActive = false
	if _, err := account.Credit(100, TransactionTypeDeposit, "", now); !errors.Is(err, ErrAccountInactive) {
		t.Errorf("inactive err = %v, want ErrAccountInactive", err)
	}
}

func TestAccountDebit(t *testing.T) {
	now := time.Now().UTC()

	newAccount := func(balance int64) *Account {
		a := NewAccount(uuid.New(), AccountTypeChecking)
		a.Balance = balance
		return a
	}

	t.Run("success", func(t *testing.T) {
		a := newAccount(1000)
		tx, err := a.Debit(300, 0, TransactionTypeWithdrawal, "", now)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if a.Balance != 700 || tx.Amount != 300 || tx.BalanceBefore != 1000 || tx.BalanceAfter != 700 {
			t.Errorf("balance = %d, tx = %+v", a.Balance, tx)
		}
	})

	t.Run("insufficient balance", func(t *testing.T) {
		a := newAccount(100)
		if _, err := a.Debit(101, 0, TransactionTypeWithdrawal, "", now); !errors.Is(err, ErrInsufficientBalance) {
			t.Errorf("err = %v, want ErrInsufficientBalance", err)
		}
		if a.Balance != 100 {
			t.Errorf("balance changed to %d", a.Balance)
		}
	})

	t.Run("daily limit", func(t *testing.T) {
		a := newAccount(DefaultDailyLimit * 2)
		if _, err := a.Debit(1, DefaultDailyLimit, TransactionTypeWithdrawal, "", now); !errors.Is(err, ErrDailyLimitExceeded) {
			t.Errorf("err = %v, want ErrDailyLimitExceeded", err)
		}
		if _, err := a.Debit(DefaultDailyLimit, 0, TransactionTypeWithdrawal, "", now); err != nil {
			t.Errorf("debit up to the limit should pass, got %v", err)
		}
	})

	t.Run("blocked account", func(t *testing.T) {
		a := newAccount(1000)
		a.BlockedAt = &now
		if _, err := a.Debit(1, 0, TransactionTypeWithdrawal, "", now); !errors.Is(err, ErrAccountInactive) {
			t.Errorf("err = %v, want ErrAccountInactive", err)
		}
	})

	t.Run("non positive amount", func(t *testing.T) {
		a := newAccount(1000)
		if _, err := a.Debit(-5, 0, TransactionTypeWithdrawal, "", now); !errors.Is(err, ErrInvalidAmount) {
			t.Errorf("err = %v, want ErrInvalidAmount", err)
		}
	})
}
