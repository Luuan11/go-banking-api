package usecase

import (
	"context"
	"fmt"
	"time"

	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/repository"
)

const dateLayout = "2006-01-02"

type CreateCustomerInput struct {
	FullName  string `json:"full_name"`
	Email     string `json:"email"`
	CPF       string `json:"cpf"`
	Phone     string `json:"phone"`
	BirthDate string `json:"birth_date"`
}

type CreateCustomer struct {
	customerRepo repository.Customer
	now          func() time.Time
}

func NewCreateCustomer(customerRepo repository.Customer) *CreateCustomer {
	return &CreateCustomer{
		customerRepo: customerRepo,
		now:          domain.Now,
	}
}

func (uc *CreateCustomer) Execute(ctx context.Context, input CreateCustomerInput) (*domain.Customer, error) {
	birthDate, err := time.Parse(dateLayout, input.BirthDate)
	if err != nil {
		return nil, domain.ErrInvalidBirthDate
	}

	customer, err := domain.NewCustomer(input.FullName, input.Email, input.CPF, input.Phone, birthDate, uc.now().UTC())
	if err != nil {
		return nil, err
	}

	if err := uc.customerRepo.Create(ctx, customer); err != nil {
		return nil, fmt.Errorf("failed to create customer: %w", err)
	}

	return customer, nil
}
