package utils

import (
	"crypto/rand"
	"fmt"
)

// GenerateRequestID returns a short random hex string suitable for a request ID.
func GenerateRequestID() string {
	b := make([]byte, 8)
	rand.Read(b) //nolint:errcheck
	return fmt.Sprintf("%x", b)
}
