// Package api exposes the REST layer: GET /services and GET /services/{serviceID}.
//
// Phase 4.
package api

import (
	"net/http"

	"github.com/openclimatefix/hexatron/backend/internal/api/controller"
	"github.com/openclimatefix/hexatron/backend/internal/api/middleware"
)

// NewRouter registers all routes and returns the root HTTP handler.
//
//	GET /services           – list all business services
//	GET /services/{id}      – get a single service with per-DAG health detail
func NewRouter() http.Handler {
	mux := http.NewServeMux()

	mux.Handle("GET /services", middleware.JSON(http.HandlerFunc(controller.ListServices)))
	mux.Handle("GET /services/", middleware.JSON(http.HandlerFunc(controller.GetService)))

	return mux
}
