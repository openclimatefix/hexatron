// Package middleware defines data structures used by the middleware layer.
package middleware

import "net/http"

// ResponseWriter wraps http.ResponseWriter to capture the HTTP status code
// written by a downstream handler. Used by the logging middleware.
type ResponseWriter struct {
	http.ResponseWriter
	StatusCode int
}

// WriteHeader captures the status code before delegating to the real writer.
func (rw *ResponseWriter) WriteHeader(code int) {
	rw.StatusCode = code
	rw.ResponseWriter.WriteHeader(code)
}
