package domain

import (
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

const MinCustomerAge = 18

type Customer struct {
	ID        uuid.UUID `json:"id"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	CPF       string    `json:"cpf"`
	Phone     string    `json:"phone,omitempty"`
	BirthDate time.Time `json:"birth_date"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewCustomer(fullName, email, cpf, phone string, birthDate, now time.Time) (*Customer, error) {
	fullName = strings.TrimSpace(fullName)
	if fullName == "" {
		return nil, ErrInvalidCustomerName
	}

	email = strings.ToLower(strings.TrimSpace(email))
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		return nil, ErrInvalidEmail
	}

	cpf = NormalizeCPF(cpf)
	if !IsValidCPF(cpf) {
		return nil, ErrInvalidCPF
	}

	if birthDate.IsZero() || birthDate.AddDate(MinCustomerAge, 0, 0).After(now) {
		return nil, ErrInvalidBirthDate
	}

	return &Customer{
		ID:        uuid.New(),
		FullName:  fullName,
		Email:     email,
		CPF:       cpf,
		Phone:     strings.TrimSpace(phone),
		BirthDate: birthDate,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}
