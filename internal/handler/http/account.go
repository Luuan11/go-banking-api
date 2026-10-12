package http

import (
	"context"
	"net/http"
	"strconv"

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

type transactionLister interface {
	Execute(ctx context.Context, input usecase.ListTransactionsInput) (*usecase.ListTransactionsOutput, error)
}

type Account struct {
	createAccountUC    accountCreator
	getAccountUC       accountGetter
	listTransactionsUC transactionLister
}

func NewAccount(createAccountUC accountCreator, getAccountUC accountGetter, listTransactionsUC transactionLister) *Account {
	return &Account{
		createAccountUC:    createAccountUC,
		getAccountUC:       getAccountUC,
		listTransactionsUC: listTransactionsUC,
	}
}

func (h *Account) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/accounts", h.Create)
	mux.HandleFunc("GET /api/accounts/{id}", h.Get)
	mux.HandleFunc("GET /api/accounts/{id}/transactions", h.ListTransactions)
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

func (h *Account) ListTransactions(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r, "id")
	if !ok {
		writeError(w, r, domain.ErrInvalidAccountID)
		return
	}

	limit, err := queryInt(r, "limit")
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_LIMIT", "limit must be an integer")
		return
	}
	offset, err := queryInt(r, "offset")
	if err != nil {
		Error(w, http.StatusBadRequest, "INVALID_OFFSET", "offset must be an integer")
		return
	}

	output, err := h.listTransactionsUC.Execute(r.Context(), usecase.ListTransactionsInput{
		AccountID: id,
		Limit:     limit,
		Offset:    offset,
	})
	if err != nil {
		writeError(w, r, err)
		return
	}

	JSON(w, http.StatusOK, output)
}

func queryInt(r *http.Request, key string) (int, error) {
	value := r.URL.Query().Get(key)
	if value == "" {
		return 0, nil
	}
	return strconv.Atoi(value)
}
