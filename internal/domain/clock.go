package domain

import "time"

// Now is UTC at Postgres precision (µs), so written and read values match.
func Now() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}
