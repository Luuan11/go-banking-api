package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/usecase"
)

type balanceMover interface {
	Execute(ctx context.Context, accountID uuid.UUID, input usecase.MovementInput) (*domain.Transaction, error)
}

// Operation handles the endpoints that move money.
type Operation struct {
	depositUC  balanceMover
	withdrawUC balanceMover
}

func NewOperation(depositUC, withdrawUC balanceMover) *Operation {
	return &Operation{
		depositUC:  depositUC,
		withdrawUC: withdrawUC,
	}
}

func (h *Operation) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/accounts/{id}/deposits", h.movement(h.depositUC))
	mux.HandleFunc("POST /api/accounts/{id}/withdrawals", h.movement(h.withdrawUC))
}

func (h *Operation) movement(uc balanceMover) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		accountID, ok := pathID(r, "id")
		if !ok {
			writeError(w, r, domain.ErrInvalidAccountID)
			return
		}

		var input usecase.MovementInput
		if err := decodeJSON(w, r, &input); err != nil {
			writeError(w, r, err)
			return
		}

		transaction, err := uc.Execute(r.Context(), accountID, input)
		if err != nil {
			writeError(w, r, err)
			return
		}

		JSON(w, http.StatusCreated, transaction)
	}
}
