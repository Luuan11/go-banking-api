package domain

import "time"

// Now returns the current time in UTC truncated to microseconds, the
// precision PostgreSQL stores. This way a value returned right after a write
// is identical to the one read back later.
func Now() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}
