package http

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/usecase"
)

type Account struct {
	createAccountUC *usecase.CreateAccount
}

func NewAccount(createAccountUC *usecase.CreateAccount) *Account {
	return &Account{
		createAccountUC: createAccountUC,
	}
}

func (h *Account) Create(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		Error(w, http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method not allowed")
		return
	}

	var input usecase.CreateAccountInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		Error(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	output, err := h.createAccountUC.Execute(r.Context(), input)
	if err != nil {
		h.handleError(w, err)
		return
	}

	JSON(w, http.StatusCreated, output.Account)
}

func (h *Account) handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidAccountType):
		Error(w, http.StatusBadRequest, "INVALID_ACCOUNT_TYPE", err.Error())
	case errors.Is(err, domain.ErrInvalidCustomerID):
		Error(w, http.StatusBadRequest, "INVALID_CUSTOMER_ID", err.Error())
	case errors.Is(err, domain.ErrAccountAlreadyExists):
		Error(w, http.StatusConflict, "ACCOUNT_EXISTS", err.Error())
	default:
		Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
	}
}
