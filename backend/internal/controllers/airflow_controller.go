// Package controllers handles incoming HTTP requests and delegates to the
// service layer. Each function maps 1:1 to a REST endpoint.
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

// ListServices handles GET /services.
//
// Response: structures/responses.ServiceListResponse (JSON array)
//
// Optional query params:
//   - ?search=<string>   case-insensitive name filter
//   - ?category=<string> exact category match
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

// GetService handles GET /services/{id}.
//
// Response: structures/responses.ServiceDetailResponse (JSON object)
//
// Returns 404 if the service is not found.
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

// writeUpstreamError converts an Airflow failure into a response. The detail
// stays in the log; the client gets enough to know whose fault it is.
func writeUpstreamError(w http.ResponseWriter, r *http.Request, err error) {
	log.Printf("%s %s: %v", r.Method, r.URL.Path, err)

	if clients.IsUnauthorized(err) {
		utils.WriteError(w, http.StatusBadGateway, constants.ErrAirflowUnauthorized)
		return
	}
	utils.WriteError(w, http.StatusBadGateway, constants.ErrAirflowUnreachable)
}

