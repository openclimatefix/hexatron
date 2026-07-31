package routes

import (
	"net/http"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/controllers"
)

// RegisterHealthRoutes attaches the health check endpoint to mux.
//
//	GET /health → controllers.HealthCheck
func RegisterHealthRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+constants.HealthPath, controllers.HealthCheck)
}
