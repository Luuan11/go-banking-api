package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/usecase"
)

type accountCreator interface {
	Execute(ctx context.Context, input usecase.CreateAccountInput) (*usecase.CreateAccountOutput, error)
}

type accountGetter interface {
	Execute(ctx context.Context, id uuid.UUID) (*domain.Account, error)
}

type Account struct {
	createAccountUC accountCreator
	getAccountUC    accountGetter
}

func NewAccount(createAccountUC accountCreator, getAccountUC accountGetter) *Account {
	return &Account{
		createAccountUC: createAccountUC,
		getAccountUC:    getAccountUC,
	}
}

func (h *Account) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/accounts", h.Create)
	mux.HandleFunc("GET /api/accounts/{id}", h.Get)
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

func (h *Account) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, r, domain.ErrInvalidAccountID)
		return
	}

	account, err := h.getAccountUC.Execute(r.Context(), id)
	if err != nil {
		writeError(w, r, err)
		return
	}

	JSON(w, http.StatusOK, account)
}
