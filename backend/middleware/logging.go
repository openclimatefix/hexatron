package middleware

import (
	"log/slog"
	"net/http"
	"time"

	mwstructs "github.com/openclimatefix/hexatron/backend/structures/middleware"
)

// Logging logs method, path, status code, and latency for every request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &mwstructs.ResponseWriter{ResponseWriter: w, StatusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		slog.Info("request",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.StatusCode,
			"duration", time.Since(start).String(),
			"request_id", w.Header().Get("X-Request-ID"),
		)
	})
}
