package domain

import "errors"

var (
	ErrInvalidCustomerName   = errors.New("full name is required")
	ErrInvalidEmail          = errors.New("invalid email")
	ErrInvalidCPF            = errors.New("invalid cpf")
	ErrInvalidBirthDate      = errors.New("invalid birth date: expected YYYY-MM-DD and at least 18 years old")
	ErrCustomerNotFound      = errors.New("customer not found")
	ErrCustomerAlreadyExists = errors.New("customer with this email or cpf already exists")
	ErrInvalidAccountType    = errors.New("invalid account type")
	ErrInvalidCustomerID     = errors.New("invalid customer id")
	ErrInvalidAccountID      = errors.New("invalid account id")
	ErrAccountNotFound       = errors.New("account not found")
	ErrAccountAlreadyExists  = errors.New("account already exists")
	ErrInvalidAccountNumber  = errors.New("invalid account number")
	ErrInsufficientBalance   = errors.New("insufficient balance")
	ErrAccountInactive       = errors.New("account inactive")
)
