// Package routes registers all HTTP routes and applies the middleware stack.
package routes

import (
	"net/http"

	"github.com/openclimatefix/hexatron/backend/middleware"
)

// NewRouter creates and returns the root HTTP handler with all routes and
// middleware applied.
//
// Middleware stack (outermost → innermost):
//
//	CORS → RequestID → Logging → Recovery → mux
func NewRouter() http.Handler {
	mux := http.NewServeMux()

	RegisterAirflowRoutes(mux)
	RegisterHealthRoutes(mux)

	// Apply middleware stack – outermost runs first.
	var handler http.Handler = mux
	handler = middleware.Recovery(handler)
	handler = middleware.Logging(handler)
	handler = middleware.RequestID(handler)
	handler = middleware.CORS(handler)

	return handler
}
