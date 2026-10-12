package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/luuan11/bank-system/internal/domain"
	"github.com/luuan11/bank-system/internal/middleware"
)

type errorMapping struct {
	err    error
	status int
	code   string
}

var errorMappings = []errorMapping{
	{errInvalidBody, http.StatusBadRequest, "INVALID_REQUEST"},
	{domain.ErrInvalidCustomerName, http.StatusBadRequest, "INVALID_FULL_NAME"},
	{domain.ErrInvalidEmail, http.StatusBadRequest, "INVALID_EMAIL"},
	{domain.ErrInvalidCPF, http.StatusBadRequest, "INVALID_CPF"},
	{domain.ErrInvalidBirthDate, http.StatusBadRequest, "INVALID_BIRTH_DATE"},
	{domain.ErrInvalidAccountType, http.StatusBadRequest, "INVALID_ACCOUNT_TYPE"},
	{domain.ErrInvalidCustomerID, http.StatusBadRequest, "INVALID_CUSTOMER_ID"},
	{domain.ErrInvalidAccountID, http.StatusBadRequest, "INVALID_ACCOUNT_ID"},
	{domain.ErrInvalidAmount, http.StatusBadRequest, "INVALID_AMOUNT"},
	{domain.ErrInvalidDescription, http.StatusBadRequest, "INVALID_DESCRIPTION"},
	{domain.ErrSameAccountTransfer, http.StatusBadRequest, "SAME_ACCOUNT_TRANSFER"},
	{domain.ErrCustomerNotFound, http.StatusNotFound, "CUSTOMER_NOT_FOUND"},
	{domain.ErrAccountNotFound, http.StatusNotFound, "ACCOUNT_NOT_FOUND"},
	{domain.ErrCustomerAlreadyExists, http.StatusConflict, "CUSTOMER_EXISTS"},
	{domain.ErrAccountAlreadyExists, http.StatusConflict, "ACCOUNT_EXISTS"},
	{domain.ErrInsufficientBalance, http.StatusUnprocessableEntity, "INSUFFICIENT_BALANCE"},
	{domain.ErrDailyLimitExceeded, http.StatusUnprocessableEntity, "DAILY_LIMIT_EXCEEDED"},
	{domain.ErrAccountInactive, http.StatusUnprocessableEntity, "ACCOUNT_INACTIVE"},
}

var internalError = errorMapping{nil, http.StatusInternalServerError, "INTERNAL_ERROR"}

func classify(err error) errorMapping {
	for _, m := range errorMappings {
		if errors.Is(err, m.err) {
			return m
		}
	}
	return internalError
}

// writeError maps domain errors to HTTP; unknown errors become a logged 500.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	m := classify(err)
	middleware.SetErrorCode(r.Context(), m.code)

	if m.err == nil {
		slog.ErrorContext(r.Context(), "unhandled error", "method", r.Method, "path", r.URL.Path, "error", err)
		Error(w, m.status, m.code, "Internal server error")
		return
	}
	Error(w, m.status, m.code, m.err.Error())
}
