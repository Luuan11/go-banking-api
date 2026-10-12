package repository

import "context"

type Repositories struct {
	Accounts     Account
	Transactions Transaction
}

// UnitOfWork runs fn in one transaction, rolled back if fn fails.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context, repos Repositories) error) error
}
