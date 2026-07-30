package utils

import (
	"crypto/rand"
	"fmt"
	"strings"
)

// ContainsInsensitive returns true if s contains substr (case-insensitive).
func ContainsInsensitive(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// EqualInsensitive returns true if a and b are equal (case-insensitive).
func EqualInsensitive(a, b string) bool {
	return strings.EqualFold(a, b)
}

// GenerateRequestID returns a short random hex string suitable for a request ID.
func GenerateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	return fmt.Sprintf("%x", b)
}
