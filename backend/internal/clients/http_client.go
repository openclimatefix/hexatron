package clients

import (
	"net/http"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
)

// NewHTTPClient returns a pre-configured *http.Client.
// Timeout is sourced from constants.DefaultHTTPTimeout.
func NewHTTPClient() *http.Client {
	return &http.Client{Timeout: constants.DefaultHTTPTimeout}
}
