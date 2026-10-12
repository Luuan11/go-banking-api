package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
)

type Customer struct {
	db DBTX
}

func NewCustomer(db DBTX) *Customer {
	return &Customer{db: db}
}

func (r *Customer) Create(ctx context.Context, customer *domain.Customer) error {
	query := `
		INSERT INTO customers (id, full_name, email, cpf, phone, birth_date, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NULLIF($5, ''), $6, $7, $8)
	`

	_, err := r.db.ExecContext(
		ctx,
		query,
		customer.ID,
		customer.FullName,
		customer.Email,
		customer.CPF,
		customer.Phone,
		customer.BirthDate,
		customer.CreatedAt,
		customer.UpdatedAt,
	)
	if _, ok := constraintViolation(err, uniqueViolation); ok {
		return domain.ErrCustomerAlreadyExists
	}
	return err
}

func (r *Customer) FindByID(ctx context.Context, id uuid.UUID) (*domain.Customer, error) {
	query := `
		SELECT id, full_name, email, cpf, COALESCE(phone, ''), birth_date, created_at, updated_at
		FROM customers WHERE id = $1
	`

	customer := &domain.Customer{}
	err := r.db.QueryRowContext(ctx, query, id).Scan(
		&customer.ID,
		&customer.FullName,
		&customer.Email,
		&customer.CPF,
		&customer.Phone,
		&customer.BirthDate,
		&customer.CreatedAt,
		&customer.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, err
	}

	customer.BirthDate = customer.BirthDate.UTC()
	customer.CreatedAt = customer.CreatedAt.UTC()
	customer.UpdatedAt = customer.UpdatedAt.UTC()
	return customer, nil
}
