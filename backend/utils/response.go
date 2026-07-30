// Package utils provides shared helper functions used across all layers.
package utils

import (
	"encoding/json"
	"net/http"

	"github.com/openclimatefix/hexatron/backend/constants"
	"github.com/openclimatefix/hexatron/backend/structures/responses"
)

// WriteJSON encodes v as JSON and writes it to w with the given HTTP status code.
func WriteJSON(w http.ResponseWriter, statusCode int, v any) {
	w.Header().Set(constants.HeaderContentType, constants.ContentTypeJSON)
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(v)
}

// WriteError writes a JSON error response with the given status code and message.
func WriteError(w http.ResponseWriter, statusCode int, message string) {
	WriteJSON(w, statusCode, responses.ErrorResponse{Error: message})
}
