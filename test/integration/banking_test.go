// Package integration runs against TEST_DATABASE_URL (skipped if unset).
// WARNING: it truncates every table.
package integration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math/rand/v2"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/repository/postgres"
	"github.com/luuan11/bank-system/internal/usecase"
	"github.com/luuan11/bank-system/pkg/database"
)

var ctx = context.Background()

type app struct {
	db               *sql.DB
	createCustomer   *usecase.CreateCustomer
	createAccount    *usecase.CreateAccount
	deposit          *usecase.Deposit
	withdraw         *usecase.Withdraw
	transfer         *usecase.Transfer
	listTransactions *usecase.ListTransactions
	accounts         *postgres.Account
}

func setup(t *testing.T) *app {
	t.Helper()

	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	db, err := database.Connect(ctx, url, database.Options{})
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	// Twice, to check idempotency.
	for range 2 {
		if err := database.Migrate(ctx, db); err != nil {
			t.Fatalf("migrate: %v", err)
		}
	}

	if _, err := db.ExecContext(ctx, `TRUNCATE transactions, accounts, customers CASCADE`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	customers := postgres.NewCustomer(db)
	accounts := postgres.NewAccount(db)
	transactions := postgres.NewTransaction(db)
	uow := postgres.NewUnitOfWork(db)

	return &app{
		db:               db,
		createCustomer:   usecase.NewCreateCustomer(customers),
		createAccount:    usecase.NewCreateAccount(accounts, customers),
		deposit:          usecase.NewDeposit(uow),
		withdraw:         usecase.NewWithdraw(uow),
		transfer:         usecase.NewTransfer(uow),
		listTransactions: usecase.NewListTransactions(accounts, transactions),
		accounts:         accounts,
	}
}

func randomCPF() string {
	digits := make([]int, 11)
	for i := range 9 {
		digits[i] = rand.IntN(10)
	}
	for _, n := range []int{9, 10} {
		sum := 0
		for i := range n {
			sum += digits[i] * (n + 1 - i)
		}
		digits[n] = (sum * 10) % 11 % 10
	}

	cpf := ""
	for _, d := range digits {
		cpf += fmt.Sprint(d)
	}
	return cpf
}

func (a *app) newCustomer(t *testing.T) *domain.Customer {
	t.Helper()
	customer, err := a.createCustomer.Execute(ctx, usecase.CreateCustomerInput{
		FullName:  "Test Customer",
		Email:     uuid.NewString() + "@example.com",
		CPF:       randomCPF(),
		BirthDate: "1990-01-01",
	})
	if err != nil {
		t.Fatalf("create customer: %v", err)
	}
	return customer
}

func (a *app) newAccount(t *testing.T, balance int64) *domain.Account {
	t.Helper()
	out, err := a.createAccount.Execute(ctx, usecase.CreateAccountInput{
		CustomerID: a.newCustomer(t).ID,
		Type:       domain.AccountTypeChecking,
	})
	if err != nil {
		t.Fatalf("create account: %v", err)
	}
	if balance > 0 {
		if _, err := a.deposit.Execute(ctx, out.Account.ID, usecase.MovementInput{Amount: balance}); err != nil {
			t.Fatalf("deposit: %v", err)
		}
	}
	return out.Account
}

func (a *app) balance(t *testing.T, id uuid.UUID) int64 {
	t.Helper()
	account, err := a.accounts.FindByID(ctx, id)
	if err != nil {
		t.Fatalf("find account: %v", err)
	}
	return account.Balance
}

func TestCustomerAndAccountConstraints(t *testing.T) {
	a := setup(t)
	customer := a.newCustomer(t)

	_, err := a.createCustomer.Execute(ctx, usecase.CreateCustomerInput{
		FullName:  "Someone Else",
		Email:     "other@example.com",
		CPF:       customer.CPF,
		BirthDate: "1990-01-01",
	})
	if !errors.Is(err, domain.ErrCustomerAlreadyExists) {
		t.Errorf("duplicate cpf err = %v, want ErrCustomerAlreadyExists", err)
	}

	for _, accountType := range []domain.AccountType{domain.AccountTypeChecking, domain.AccountTypeSavings} {
		if _, err := a.createAccount.Execute(ctx, usecase.CreateAccountInput{CustomerID: customer.ID, Type: accountType}); err != nil {
			t.Fatalf("create %s account: %v", accountType, err)
		}
	}

	// Skip the use case pre-check, like a concurrent request would.
	duplicate := domain.NewAccount(customer.ID, domain.AccountTypeChecking)
	duplicate.Number = "999999999999"
	if err := a.accounts.Create(ctx, duplicate); !errors.Is(err, domain.ErrAccountAlreadyExists) {
		t.Errorf("constraint err = %v, want ErrAccountAlreadyExists", err)
	}

	orphan := domain.NewAccount(uuid.New(), domain.AccountTypeChecking)
	orphan.Number = "888888888888"
	if err := a.accounts.Create(ctx, orphan); !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Errorf("foreign key err = %v, want ErrCustomerNotFound", err)
	}
}

func TestMoneyFlow(t *testing.T) {
	a := setup(t)
	from := a.newAccount(t, 10000)
	to := a.newAccount(t, 0)

	if _, err := a.withdraw.Execute(ctx, from.ID, usecase.MovementInput{Amount: 1000, Description: "atm"}); err != nil {
		t.Fatalf("withdraw: %v", err)
	}

	out, err := a.transfer.Execute(ctx, usecase.TransferInput{FromAccountID: from.ID, ToAccountID: to.ID, Amount: 4000})
	if err != nil {
		t.Fatalf("transfer: %v", err)
	}

	if got := a.balance(t, from.ID); got != 5000 {
		t.Errorf("from balance = %d, want 5000", got)
	}
	if got := a.balance(t, to.ID); got != 4000 {
		t.Errorf("to balance = %d, want 4000", got)
	}

	_, err = a.transfer.Execute(ctx, usecase.TransferInput{FromAccountID: from.ID, ToAccountID: to.ID, Amount: 5001})
	if !errors.Is(err, domain.ErrInsufficientBalance) {
		t.Errorf("err = %v, want ErrInsufficientBalance", err)
	}

	statement, err := a.listTransactions.Execute(ctx, usecase.ListTransactionsInput{AccountID: from.ID})
	if err != nil {
		t.Fatalf("list: %v", err)
	}

	// Write and read responses must match.
	stored, err := a.accounts.FindByID(ctx, from.ID)
	if err != nil {
		t.Fatalf("find account: %v", err)
	}
	if !stored.CreatedAt.Equal(from.CreatedAt) || stored.CreatedAt.Location() != time.UTC {
		t.Errorf("created_at = %v, want %v (UTC)", stored.CreatedAt, from.CreatedAt)
	}
	if !stored.UpdatedAt.Equal(out.Debit.CreatedAt) {
		t.Errorf("updated_at = %v, want %v", stored.UpdatedAt, out.Debit.CreatedAt)
	}
	if got := statement.Items[0].CreatedAt; !got.Equal(out.Debit.CreatedAt) || got.Location() != time.UTC {
		t.Errorf("statement created_at = %v, want %v", got, out.Debit.CreatedAt)
	}
	if statement.Total != 3 {
		t.Fatalf("total = %d, want 3 (deposit, withdrawal, transfer_out)", statement.Total)
	}
	latest := statement.Items[0]
	if latest.Type != domain.TransactionTypeTransferOut || latest.ReferenceID == nil || *latest.ReferenceID != out.ID {
		t.Errorf("latest transaction = %+v", latest)
	}
	if statement.Items[1].Description != "atm" {
		t.Errorf("description = %q, want atm", statement.Items[1].Description)
	}
}

func TestConcurrentWithdrawalsNeverOverdraw(t *testing.T) {
	a := setup(t)
	account := a.newAccount(t, 500)

	const workers = 20
	var (
		wg        sync.WaitGroup
		mu        sync.Mutex
		succeeded int
	)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := a.withdraw.Execute(ctx, account.ID, usecase.MovementInput{Amount: 50})
			switch {
			case err == nil:
				mu.Lock()
				succeeded++
				mu.Unlock()
			case !errors.Is(err, domain.ErrInsufficientBalance):
				t.Errorf("unexpected error: %v", err)
			}
		}()
	}
	wg.Wait()

	if succeeded != 10 {
		t.Errorf("succeeded = %d, want 10", succeeded)
	}
	if got := a.balance(t, account.ID); got != 0 {
		t.Errorf("balance = %d, want 0", got)
	}
}

func TestConcurrentOppositeTransfersDoNotDeadlock(t *testing.T) {
	a := setup(t)
	first := a.newAccount(t, 10000)
	second := a.newAccount(t, 10000)

	const workers = 20
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			input := usecase.TransferInput{FromAccountID: first.ID, ToAccountID: second.ID, Amount: 100}
			if i%2 == 1 {
				input.FromAccountID, input.ToAccountID = second.ID, first.ID
			}
			if _, err := a.transfer.Execute(ctx, input); err != nil {
				t.Errorf("transfer %d: %v", i, err)
			}
		}()
	}
	wg.Wait()

	total := a.balance(t, first.ID) + a.balance(t, second.ID)
	if total != 20000 {
		t.Errorf("total balance = %d, want 20000", total)
	}
}
