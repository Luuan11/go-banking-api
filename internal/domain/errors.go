package domain

import "errors"

var (
	ErrInvalidAccountType   = errors.New("invalid account type")
	ErrInvalidCustomerID    = errors.New("invalid customer id")
	ErrAccountNotFound      = errors.New("account not found")
	ErrAccountAlreadyExists = errors.New("account already exists")
	ErrInvalidAccountNumber = errors.New("invalid account number")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrAccountInactive      = errors.New("account inactive")
)
