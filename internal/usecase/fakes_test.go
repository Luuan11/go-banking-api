package usecase

import (
	"context"
	"maps"
	"slices"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/repository"
)

// memoryStore fakes postgres; Do() restores a snapshot on error (rollback).
type memoryStore struct {
	mu           sync.Mutex
	customers    map[uuid.UUID]domain.Customer
	accounts     map[uuid.UUID]domain.Account
	transactions []domain.Transaction

	// takenNumbers forces account number collisions.
	takenNumbers int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{
		customers: map[uuid.UUID]domain.Customer{},
		accounts:  map[uuid.UUID]domain.Account{},
	}
}

func (s *memoryStore) Do(ctx context.Context, fn func(ctx context.Context, repos repository.Repositories) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	accounts := maps.Clone(s.accounts)
	transactions := slices.Clone(s.transactions)

	repos := repository.Repositories{
		Accounts:     &memoryAccounts{s: s, locked: true},
		Transactions: &memoryTransactions{s: s, locked: true},
	}
	if err := fn(ctx, repos); err != nil {
		s.accounts = accounts
		s.transactions = transactions
		return err
	}
	return nil
}

func (s *memoryStore) addCustomer() domain.Customer {
	c := domain.Customer{ID: uuid.New(), FullName: "Test", Email: uuid.NewString() + "@example.com"}
	s.customers[c.ID] = c
	return c
}

func (s *memoryStore) addAccount(balance int64) domain.Account {
	a := domain.NewAccount(uuid.New(), domain.AccountTypeChecking)
	a.Balance = balance
	a.Number = uuid.NewString()[:12]
	s.accounts[a.ID] = *a
	return *a
}

type memoryCustomers struct{ s *memoryStore }

func (r *memoryCustomers) Create(_ context.Context, c *domain.Customer) error {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	for _, existing := range r.s.customers {
		if existing.Email == c.Email || existing.CPF == c.CPF {
			return domain.ErrCustomerAlreadyExists
		}
	}
	r.s.customers[c.ID] = *c
	return nil
}

func (r *memoryCustomers) FindByID(_ context.Context, id uuid.UUID) (*domain.Customer, error) {
	r.s.mu.Lock()
	defer r.s.mu.Unlock()
	c, ok := r.s.customers[id]
	if !ok {
		return nil, domain.ErrCustomerNotFound
	}
	return &c, nil
}

// locked is true inside Do(), which already holds the store lock.
type memoryAccounts struct {
	s      *memoryStore
	locked bool
}

func (r *memoryAccounts) lock() func() {
	if r.locked {
		return func() {}
	}
	r.s.mu.Lock()
	return r.s.mu.Unlock
}

func (r *memoryAccounts) Create(_ context.Context, a *domain.Account) error {
	defer r.lock()()
	if r.s.takenNumbers > 0 {
		r.s.takenNumbers--
		return repository.ErrAccountNumberTaken
	}
	for _, existing := range r.s.accounts {
		if existing.CustomerID == a.CustomerID && existing.Type == a.Type {
			return domain.ErrAccountAlreadyExists
		}
	}
	r.s.accounts[a.ID] = *a
	return nil
}

func (r *memoryAccounts) FindByID(_ context.Context, id uuid.UUID) (*domain.Account, error) {
	defer r.lock()()
	a, ok := r.s.accounts[id]
	if !ok {
		return nil, domain.ErrAccountNotFound
	}
	return &a, nil
}

func (r *memoryAccounts) FindByIDForUpdate(ctx context.Context, id uuid.UUID) (*domain.Account, error) {
	return r.FindByID(ctx, id)
}

func (r *memoryAccounts) UpdateBalance(_ context.Context, a *domain.Account) error {
	defer r.lock()()
	existing, ok := r.s.accounts[a.ID]
	if !ok {
		return domain.ErrAccountNotFound
	}
	existing.Balance = a.Balance
	existing.UpdatedAt = a.UpdatedAt
	r.s.accounts[a.ID] = existing
	return nil
}

func (r *memoryAccounts) FindByNumber(_ context.Context, number string) (*domain.Account, error) {
	defer r.lock()()
	for _, a := range r.s.accounts {
		if a.Number == number {
			return &a, nil
		}
	}
	return nil, domain.ErrAccountNotFound
}

func (r *memoryAccounts) ExistsByCustomerAndType(_ context.Context, customerID uuid.UUID, t domain.AccountType) (bool, error) {
	defer r.lock()()
	for _, a := range r.s.accounts {
		if a.CustomerID == customerID && a.Type == t {
			return true, nil
		}
	}
	return false, nil
}

type memoryTransactions struct {
	s      *memoryStore
	locked bool
}

func (r *memoryTransactions) lock() func() {
	if r.locked {
		return func() {}
	}
	r.s.mu.Lock()
	return r.s.mu.Unlock
}

func (r *memoryTransactions) Create(_ context.Context, t *domain.Transaction) error {
	defer r.lock()()
	r.s.transactions = append(r.s.transactions, *t)
	return nil
}

func (r *memoryTransactions) SumDebitsSince(_ context.Context, accountID uuid.UUID, since time.Time) (int64, error) {
	defer r.lock()()
	var total int64
	for _, t := range r.s.transactions {
		isDebit := t.Type == domain.TransactionTypeWithdrawal || t.Type == domain.TransactionTypeTransferOut
		if t.AccountID == accountID && isDebit && !t.CreatedAt.Before(since) {
			total += t.Amount
		}
	}
	return total, nil
}

func (r *memoryTransactions) ListByAccount(_ context.Context, accountID uuid.UUID, limit, offset int) ([]domain.Transaction, int, error) {
	defer r.lock()()
	var all []domain.Transaction
	for i := len(r.s.transactions) - 1; i >= 0; i-- {
		if r.s.transactions[i].AccountID == accountID {
			all = append(all, r.s.transactions[i])
		}
	}
	total := len(all)
	if offset >= total {
		return []domain.Transaction{}, total, nil
	}
	end := min(offset+limit, total)
	return all[offset:end], total, nil
}
