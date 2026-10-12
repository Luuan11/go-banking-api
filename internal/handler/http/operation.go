package http

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/metrics"
	"github.com/luuan11/bank-system/internal/usecase"
)

type balanceMover interface {
	Execute(ctx context.Context, accountID uuid.UUID, input usecase.MovementInput) (*domain.Transaction, error)
}

type transferer interface {
	Execute(ctx context.Context, input usecase.TransferInput) (*usecase.TransferOutput, error)
}

type operationRecorder interface {
	Operation(operation string, amount int64, result string)
}

type Operation struct {
	depositUC  balanceMover
	withdrawUC balanceMover
	transferUC transferer
	recorder   operationRecorder
}

func NewOperation(depositUC, withdrawUC balanceMover, transferUC transferer, recorder operationRecorder) *Operation {
	return &Operation{
		depositUC:  depositUC,
		withdrawUC: withdrawUC,
		transferUC: transferUC,
		recorder:   recorder,
	}
}

// record is only called once the use case ran; malformed requests are skipped.
func (h *Operation) record(operation string, amount int64, err error) {
	result := metrics.ResultOK
	if err != nil {
		result = classify(err).code
	}
	h.recorder.Operation(operation, amount, result)
}

func (h *Operation) Register(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/accounts/{id}/deposits", h.movement("deposit", h.depositUC))
	mux.HandleFunc("POST /api/accounts/{id}/withdrawals", h.movement("withdrawal", h.withdrawUC))
	mux.HandleFunc("POST /api/transfers", h.Transfer)
}

func (h *Operation) Transfer(w http.ResponseWriter, r *http.Request) {
	var input usecase.TransferInput
	if err := decodeJSON(w, r, &input); err != nil {
		writeError(w, r, err)
		return
	}

	output, err := h.transferUC.Execute(r.Context(), input)
	h.record("transfer", input.Amount, err)
	if err != nil {
		writeError(w, r, err)
		return
	}

	JSON(w, http.StatusCreated, output)
}

func (h *Operation) movement(operation string, uc balanceMover) http.HandlerFunc {
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
		h.record(operation, input.Amount, err)
		if err != nil {
			writeError(w, r, err)
			return
		}

		JSON(w, http.StatusCreated, transaction)
	}
}
