package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/luuan11/bank-system/internal/domain"
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
	{domain.ErrCustomerNotFound, http.StatusNotFound, "CUSTOMER_NOT_FOUND"},
	{domain.ErrAccountNotFound, http.StatusNotFound, "ACCOUNT_NOT_FOUND"},
	{domain.ErrCustomerAlreadyExists, http.StatusConflict, "CUSTOMER_EXISTS"},
	{domain.ErrAccountAlreadyExists, http.StatusConflict, "ACCOUNT_EXISTS"},
}

// writeError translates domain errors into HTTP responses. Anything unknown
// is logged and returned as a generic 500 so internals don't leak.
func writeError(w http.ResponseWriter, r *http.Request, err error) {
	for _, m := range errorMappings {
		if errors.Is(err, m.err) {
			Error(w, m.status, m.code, m.err.Error())
			return
		}
	}

	slog.ErrorContext(r.Context(), "unhandled error", "method", r.Method, "path", r.URL.Path, "error", err)
	Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "Internal server error")
}
