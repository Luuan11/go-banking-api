package postgres

import (
	"errors"

	"github.com/lib/pq"
)

const (
	uniqueViolation     = "23505"
	foreignKeyViolation = "23503"
)

// constraintViolation reports whether err is a postgres error with the given
// code and returns the name of the violated constraint.
func constraintViolation(err error, code string) (string, bool) {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && string(pqErr.Code) == code {
		return pqErr.Constraint, true
	}
	return "", false
}
