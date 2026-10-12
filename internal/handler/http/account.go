package http

import (
	"context"
	"net/http"

	"github.com/luuan11/bank-system/internal/usecase"
)

type accountCreator interface {
	Execute(ctx context.Context, input usecase.CreateAccountInput) (*usecase.CreateAccountOutput, error)
}

type Account struct {
	createAccountUC accountCreator
}

func NewAccount(createAccountUC accountCreator) *Account {
	return &Account{
		createAccountUC: createAccountUC,
	}
}

func (h *Account) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/accounts", h.Create)
}

func (h *Account) Create(w http.ResponseWriter, r *http.Request) {
	var input usecase.CreateAccountInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, r, err)
		return
	}

	output, err := h.createAccountUC.Execute(r.Context(), input)
	if err != nil {
		writeError(w, r, err)
		return
	}

	JSON(w, http.StatusCreated, output.Account)
}
