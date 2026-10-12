package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
)

var ctx = context.Background()

func TestCreateCustomer(t *testing.T) {
	store := newMemoryStore()
	uc := NewCreateCustomer(&memoryCustomers{s: store})

	input := CreateCustomerInput{
		FullName:  "Maria Silva",
		Email:     "maria@example.com",
		CPF:       "529.982.247-25",
		BirthDate: "1995-04-20",
	}

	customer, err := uc.Execute(ctx, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if customer.CPF != "52998224725" {
		t.Errorf("cpf = %q", customer.CPF)
	}

	if _, err := uc.Execute(ctx, input); !errors.Is(err, domain.ErrCustomerAlreadyExists) {
		t.Errorf("duplicate err = %v, want ErrCustomerAlreadyExists", err)
	}

	input.BirthDate = "20/04/1995"
	if _, err := uc.Execute(ctx, input); !errors.Is(err, domain.ErrInvalidBirthDate) {
		t.Errorf("bad date err = %v, want ErrInvalidBirthDate", err)
	}
}

func TestCreateAccount(t *testing.T) {
	store := newMemoryStore()
	customer := store.addCustomer()
	uc := NewCreateAccount(&memoryAccounts{s: store}, &memoryCustomers{s: store})

	t.Run("unknown customer", func(t *testing.T) {
		_, err := uc.Execute(ctx, CreateAccountInput{CustomerID: uuid.New(), Type: domain.AccountTypeChecking})
		if !errors.Is(err, domain.ErrCustomerNotFound) {
			t.Errorf("err = %v, want ErrCustomerNotFound", err)
		}
	})

	t.Run("invalid type", func(t *testing.T) {
		_, err := uc.Execute(ctx, CreateAccountInput{CustomerID: customer.ID, Type: "crypto"})
		if !errors.Is(err, domain.ErrInvalidAccountType) {
			t.Errorf("err = %v, want ErrInvalidAccountType", err)
		}
	})

	t.Run("one account per type", func(t *testing.T) {
		for _, accountType := range []domain.AccountType{domain.AccountTypeChecking, domain.AccountTypeSavings} {
			out, err := uc.Execute(ctx, CreateAccountInput{CustomerID: customer.ID, Type: accountType})
			if err != nil {
				t.Fatalf("create %s: %v", accountType, err)
			}
			if len(out.Account.Number) != 12 {
				t.Errorf("account number = %q", out.Account.Number)
			}
		}

		_, err := uc.Execute(ctx, CreateAccountInput{CustomerID: customer.ID, Type: domain.AccountTypeChecking})
		if !errors.Is(err, domain.ErrAccountAlreadyExists) {
			t.Errorf("err = %v, want ErrAccountAlreadyExists", err)
		}
	})

	t.Run("retries account number collisions", func(t *testing.T) {
		other := store.addCustomer()
		store.takenNumbers = maxAccountNumberAttempts - 1
		if _, err := uc.Execute(ctx, CreateAccountInput{CustomerID: other.ID, Type: domain.AccountTypeChecking}); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		another := store.addCustomer()
		store.takenNumbers = maxAccountNumberAttempts
		if _, err := uc.Execute(ctx, CreateAccountInput{CustomerID: another.ID, Type: domain.AccountTypeChecking}); err == nil {
			t.Fatal("expected error after exhausting attempts")
		}
	})
}

func TestDepositAndWithdraw(t *testing.T) {
	store := newMemoryStore()
	account := store.addAccount(0)
	deposit := NewDeposit(store)
	withdraw := NewWithdraw(store)

	if _, err := deposit.Execute(ctx, account.ID, MovementInput{Amount: 5000}); err != nil {
		t.Fatalf("deposit: %v", err)
	}

	tx, err := withdraw.Execute(ctx, account.ID, MovementInput{Amount: 2000, Description: "atm"})
	if err != nil {
		t.Fatalf("withdraw: %v", err)
	}
	if tx.BalanceAfter != 3000 || tx.Type != domain.TransactionTypeWithdrawal {
		t.Errorf("unexpected transaction %+v", tx)
	}

	if _, err := withdraw.Execute(ctx, account.ID, MovementInput{Amount: 3001}); !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Errorf("err = %v, want ErrInsufficientBalance", err)
	}
	if _, err := deposit.Execute(ctx, uuid.New(), MovementInput{Amount: 1}); !errors.Is(err, domain.ErrAccountNotFound) {
		t.Errorf("err = %v, want ErrAccountNotFound", err)
	}
	if _, err := deposit.Execute(ctx, account.ID, MovementInput{Amount: -1}); !errors.Is(err, domain.ErrInvalidAmount) {
		t.Errorf("err = %v, want ErrInvalidAmount", err)
	}

	if got := store.accounts[account.ID].Balance; got != 3000 {
		t.Errorf("stored balance = %d, want 3000", got)
	}
	if got := len(store.transactions); got != 2 {
		t.Errorf("transactions = %d, want 2", got)
	}
}

func TestWithdrawDailyLimitResetsNextDay(t *testing.T) {
	store := newMemoryStore()
	account := store.addAccount(domain.DefaultDailyLimit * 3)

	now := time.Date(2026, 10, 12, 23, 0, 0, 0, time.UTC)
	withdraw := NewWithdraw(store)
	withdraw.now = func() time.Time { return now }

	if _, err := withdraw.Execute(ctx, account.ID, MovementInput{Amount: domain.DefaultDailyLimit}); err != nil {
		t.Fatalf("first withdraw: %v", err)
	}
	if _, err := withdraw.Execute(ctx, account.ID, MovementInput{Amount: 1}); !errors.Is(err, domain.ErrDailyLimitExceeded) {
		t.Fatalf("err = %v, want ErrDailyLimitExceeded", err)
	}

	now = now.Add(2 * time.Hour)
	if _, err := withdraw.Execute(ctx, account.ID, MovementInput{Amount: 1}); err != nil {
		t.Errorf("withdraw on next day: %v", err)
	}
}

func TestTransfer(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		store := newMemoryStore()
		from := store.addAccount(10000)
		to := store.addAccount(500)

		out, err := NewTransfer(store).Execute(ctx, TransferInput{FromAccountID: from.ID, ToAccountID: to.ID, Amount: 2500})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if store.accounts[from.ID].Balance != 7500 || store.accounts[to.ID].Balance != 3000 {
			t.Errorf("balances = %d / %d", store.accounts[from.ID].Balance, store.accounts[to.ID].Balance)
		}
		if *out.Debit.ReferenceID != out.ID || *out.Credit.ReferenceID != out.ID {
			t.Error("transactions must reference the transfer id")
		}
		if out.Debit.Type != domain.TransactionTypeTransferOut || out.Credit.Type != domain.TransactionTypeTransferIn {
			t.Errorf("types = %s / %s", out.Debit.Type, out.Credit.Type)
		}
	})

	t.Run("insufficient balance rolls back", func(t *testing.T) {
		store := newMemoryStore()
		from := store.addAccount(100)
		to := store.addAccount(0)

		_, err := NewTransfer(store).Execute(ctx, TransferInput{FromAccountID: from.ID, ToAccountID: to.ID, Amount: 101})
		if !errors.Is(err, domain.ErrInsufficientBalance) {
			t.Fatalf("err = %v, want ErrInsufficientBalance", err)
		}
		if store.accounts[from.ID].Balance != 100 || store.accounts[to.ID].Balance != 0 || len(store.transactions) != 0 {
			t.Error("state changed after failed transfer")
		}
	})

	t.Run("inactive destination rolls back", func(t *testing.T) {
		store := newMemoryStore()
		from := store.addAccount(100)
		to := store.addAccount(0)
		inactive := store.accounts[to.ID]
		inactive.IsActive = false
		store.accounts[to.ID] = inactive

		_, err := NewTransfer(store).Execute(ctx, TransferInput{FromAccountID: from.ID, ToAccountID: to.ID, Amount: 50})
		if !errors.Is(err, domain.ErrAccountInactive) {
			t.Fatalf("err = %v, want ErrAccountInactive", err)
		}
		if store.accounts[from.ID].Balance != 100 {
			t.Error("debit was not rolled back")
		}
	})

	t.Run("validation", func(t *testing.T) {
		store := newMemoryStore()
		a := store.addAccount(100)
		uc := NewTransfer(store)

		if _, err := uc.Execute(ctx, TransferInput{FromAccountID: a.ID, ToAccountID: a.ID, Amount: 1}); !errors.Is(err, domain.ErrSameAccountTransfer) {
			t.Errorf("err = %v, want ErrSameAccountTransfer", err)
		}
		if _, err := uc.Execute(ctx, TransferInput{FromAccountID: a.ID, ToAccountID: uuid.New(), Amount: 1}); !errors.Is(err, domain.ErrAccountNotFound) {
			t.Errorf("err = %v, want ErrAccountNotFound", err)
		}
		if _, err := uc.Execute(ctx, TransferInput{FromAccountID: a.ID, Amount: 1}); !errors.Is(err, domain.ErrInvalidAccountID) {
			t.Errorf("err = %v, want ErrInvalidAccountID", err)
		}
	})
}

func TestListTransactions(t *testing.T) {
	store := newMemoryStore()
	account := store.addAccount(0)
	deposit := NewDeposit(store)
	for i := 1; i <= 5; i++ {
		if _, err := deposit.Execute(ctx, account.ID, MovementInput{Amount: int64(i)}); err != nil {
			t.Fatal(err)
		}
	}

	uc := NewListTransactions(&memoryAccounts{s: store}, &memoryTransactions{s: store})

	out, err := uc.Execute(ctx, ListTransactionsInput{AccountID: account.ID, Limit: 2, Offset: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if out.Total != 5 || len(out.Items) != 2 || out.Items[0].Amount != 4 {
		t.Errorf("unexpected page: total=%d items=%+v", out.Total, out.Items)
	}

	out, err = uc.Execute(ctx, ListTransactionsInput{AccountID: account.ID, Limit: 1000})
	if err != nil {
		t.Fatal(err)
	}
	if out.Limit != MaxPageSize {
		t.Errorf("limit = %d, want %d", out.Limit, MaxPageSize)
	}

	if _, err := uc.Execute(ctx, ListTransactionsInput{AccountID: uuid.New()}); !errors.Is(err, domain.ErrAccountNotFound) {
		t.Errorf("err = %v, want ErrAccountNotFound", err)
	}
}

func TestGetCustomerAndAccount(t *testing.T) {
	store := newMemoryStore()
	customer := store.addCustomer()
	account := store.addAccount(42)

	gotCustomer, err := NewGetCustomer(&memoryCustomers{s: store}).Execute(ctx, customer.ID)
	if err != nil || gotCustomer.ID != customer.ID {
		t.Errorf("get customer = %+v, %v", gotCustomer, err)
	}
	if _, err := NewGetCustomer(&memoryCustomers{s: store}).Execute(ctx, uuid.New()); !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Errorf("err = %v, want ErrCustomerNotFound", err)
	}

	gotAccount, err := NewGetAccount(&memoryAccounts{s: store}).Execute(ctx, account.ID)
	if err != nil || gotAccount.Balance != 42 {
		t.Errorf("get account = %+v, %v", gotAccount, err)
	}
	if _, err := NewGetAccount(&memoryAccounts{s: store}).Execute(ctx, uuid.New()); !errors.Is(err, domain.ErrAccountNotFound) {
		t.Errorf("err = %v, want ErrAccountNotFound", err)
	}
}

func TestMovementDescriptionTooLong(t *testing.T) {
	store := newMemoryStore()
	account := store.addAccount(100)
	long := strings.Repeat("a", domain.MaxDescriptionLength+1)

	if _, err := NewDeposit(store).Execute(ctx, account.ID, MovementInput{Amount: 1, Description: long}); !errors.Is(err, domain.ErrInvalidDescription) {
		t.Errorf("deposit err = %v, want ErrInvalidDescription", err)
	}
	if _, err := NewWithdraw(store).Execute(ctx, account.ID, MovementInput{Amount: 1, Description: long}); !errors.Is(err, domain.ErrInvalidDescription) {
		t.Errorf("withdraw err = %v, want ErrInvalidDescription", err)
	}
	other := store.addAccount(0)
	if _, err := NewTransfer(store).Execute(ctx, TransferInput{FromAccountID: account.ID, ToAccountID: other.ID, Amount: 1, Description: long}); !errors.Is(err, domain.ErrInvalidDescription) {
		t.Errorf("transfer err = %v, want ErrInvalidDescription", err)
	}
}
