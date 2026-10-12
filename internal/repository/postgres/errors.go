package postgres

import (
	"errors"

	"github.com/lib/pq"
)

const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

// constraintViolation returns the violated constraint name for the given code.
func constraintViolation(err error, code string) (string, bool) {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && string(pqErr.Code) == code {
		return pqErr.Constraint, true
	}
	return "", false
}
