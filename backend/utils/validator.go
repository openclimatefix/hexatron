package utils

import "strings"

// IsEmpty returns true if s is blank (empty or only whitespace).
func IsEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}
