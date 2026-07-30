package utils

import "time"

// NowUTC returns the current time in UTC.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// FormatRFC3339 formats t as an RFC3339 string.
func FormatRFC3339(t time.Time) string {
	return t.Format(time.RFC3339)
}
