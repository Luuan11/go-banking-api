package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/usecase"
)

type stubMover struct {
	err   error
	input usecase.MovementInput
}

func (s *stubMover) Execute(_ context.Context, accountID uuid.UUID, input usecase.MovementInput) (*domain.Transaction, error) {
	s.input = input
	if s.err != nil {
		return nil, s.err
	}
	return &domain.Transaction{ID: uuid.New(), AccountID: accountID, Amount: input.Amount}, nil
}

type stubTransferer struct{ err error }

func (s stubTransferer) Execute(context.Context, usecase.TransferInput) (*usecase.TransferOutput, error) {
	return nil, s.err
}

func newTestMux(mover *stubMover, transferErr error) *http.ServeMux {
	mux := http.NewServeMux()
	NewOperation(mover, mover, stubTransferer{err: transferErr}).Register(mux)
	return mux
}

func do(t *testing.T, mux http.Handler, method, path, body string) (*httptest.ResponseRecorder, Response) {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	var resp Response
	if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}
	return rec, resp
}

func TestDepositHandler(t *testing.T) {
	accountID := uuid.NewString()

	tests := []struct {
		name       string
		path       string
		body       string
		ucErr      error
		wantStatus int
		wantCode   string
	}{
		{"created", "/api/accounts/" + accountID + "/deposits", `{"amount":100}`, nil, http.StatusCreated, ""},
		{"invalid account id", "/api/accounts/abc/deposits", `{"amount":100}`, nil, http.StatusBadRequest, "INVALID_ACCOUNT_ID"},
		{"malformed json", "/api/accounts/" + accountID + "/deposits", `{"amount":`, nil, http.StatusBadRequest, "INVALID_REQUEST"},
		{"unknown field", "/api/accounts/" + accountID + "/deposits", `{"amount":1,"amout":2}`, nil, http.StatusBadRequest, "INVALID_REQUEST"},
		{"not found", "/api/accounts/" + accountID + "/deposits", `{"amount":1}`, domain.ErrAccountNotFound, http.StatusNotFound, "ACCOUNT_NOT_FOUND"},
		{"wrapped domain error", "/api/accounts/" + accountID + "/withdrawals", `{"amount":1}`, errors.Join(errors.New("ctx"), domain.ErrInsufficientBalance), http.StatusUnprocessableEntity, "INSUFFICIENT_BALANCE"},
		{"unexpected error", "/api/accounts/" + accountID + "/deposits", `{"amount":1}`, errors.New("connection reset"), http.StatusInternalServerError, "INTERNAL_ERROR"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mover := &stubMover{err: tt.ucErr}
			rec, resp := do(t, newTestMux(mover, nil), http.MethodPost, tt.path, tt.body)

			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if tt.wantCode == "" {
				if !resp.Success || resp.Error != nil {
					t.Errorf("unexpected response %+v", resp)
				}
				return
			}
			if resp.Success || resp.Error == nil || resp.Error.Code != tt.wantCode {
				t.Errorf("error = %+v, want code %s", resp.Error, tt.wantCode)
			}
			if tt.wantStatus == http.StatusInternalServerError && strings.Contains(resp.Error.Message, "connection") {
				t.Error("internal error details leaked to the client")
			}
		})
	}
}

func TestTransferHandlerMethodNotAllowed(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/api/transfers", nil)
	rec := httptest.NewRecorder()
	newTestMux(&stubMover{}, nil).ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want 405", rec.Code)
	}
}

func TestTransferHandlerSameAccount(t *testing.T) {
	rec, resp := do(t, newTestMux(&stubMover{}, domain.ErrSameAccountTransfer), http.MethodPost, "/api/transfers", `{"amount":1}`)
	if rec.Code != http.StatusBadRequest || resp.Error.Code != "SAME_ACCOUNT_TRANSFER" {
		t.Errorf("status = %d, error = %+v", rec.Code, resp.Error)
	}
}
