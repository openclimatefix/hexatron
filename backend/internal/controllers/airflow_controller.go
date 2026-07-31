// Package controllers handles incoming HTTP requests and delegates to the service layer.
package controllers

import (
	"log"
	"net/http"

	"github.com/openclimatefix/hexatron/backend/internal/clients"
	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/services"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
	"github.com/openclimatefix/hexatron/backend/internal/utils"
)

// AirflowController handles HTTP requests for service and DAG health status.
type AirflowController struct {
	airflowService *services.AirflowService
}

// NewAirflowController constructs an AirflowController from application config.
func NewAirflowController(cfg *configstructs.Config) *AirflowController {
	return &AirflowController{
		airflowService: services.NewAirflowService(cfg),
	}
}

// ListServices handles GET /services, with optional ?search and ?category filters.
func (c *AirflowController) ListServices(w http.ResponseWriter, r *http.Request) {
	summaries, err := c.airflowService.ListServices(
		r.Context(),
		r.URL.Query().Get("search"),
		r.URL.Query().Get("category"),
	)
	if err != nil {
		writeUpstreamError(w, r, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, summaries)
}

// GetService handles GET /services/{id}, returning 404 if the service is not found.
func (c *AirflowController) GetService(w http.ResponseWriter, r *http.Request) {
	serviceID := r.PathValue("id")

	detail, found, err := c.airflowService.GetServiceByID(r.Context(), serviceID)
	if !found {
		utils.WriteError(w, http.StatusNotFound, constants.ErrServiceNotFound+": "+serviceID)
		return
	}
	if err != nil {
		writeUpstreamError(w, r, err)
		return
	}
	utils.WriteJSON(w, http.StatusOK, detail)
}

// writeUpstreamError logs an Airflow failure and converts it into a 502 response.
func writeUpstreamError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("%s %s: %v", r.Method, r.URL.Path, err)

	if clients.IsUnauthorized(err) {
		utils.WriteError(w, http.StatusBadGateway, constants.ErrAirflowUnauthorized)
		return
	}
	utils.WriteError(w, http.StatusBadGateway, constants.ErrAirflowUnreachable)
}
