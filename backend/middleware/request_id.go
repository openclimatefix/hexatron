package middleware

import (
	"context"
	"net/http"

	"github.com/openclimatefix/hexatron/backend/utils"
)

type contextKey string

const requestIDKey contextKey = "request_id"

// RequestID attaches a unique X-Request-ID header to every request and
// stores the ID in the request context.
// If the client already sends an X-Request-ID it is reused.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = utils.GenerateRequestID()
		}
		w.Header().Set("X-Request-ID", id)
		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// GetRequestID retrieves the request ID stored in the context by RequestID middleware.
func GetRequestID(r *http.Request) string {
	if id, ok := r.Context().Value(requestIDKey).(string); ok {
		return id
	}
	return ""
}
