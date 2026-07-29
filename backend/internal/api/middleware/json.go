// Package middleware provides reusable HTTP middleware for the API layer.
package middleware

import "net/http"

// JSON wraps a handler and ensures every response carries the
// Content-Type: application/json header.
func JSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		next.ServeHTTP(w, r)
	})
}
