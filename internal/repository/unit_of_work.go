package repository

import "context"

// Repositories groups the repositories that share a single database
// transaction inside a unit of work.
type Repositories struct {
	Accounts     Account
	Transactions Transaction
}

// UnitOfWork runs fn atomically: everything done through repos is committed
// if fn returns nil and rolled back otherwise.
type UnitOfWork interface {
	Do(ctx context.Context, fn func(ctx context.Context, repos Repositories) error) error
}
