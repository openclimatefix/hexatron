// Package controllers handles incoming HTTP requests and delegates to the
// service layer. Each function maps 1:1 to a REST endpoint.
package controllers

import (
	"net/http"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/models"
	"github.com/openclimatefix/hexatron/backend/internal/services"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
	"github.com/openclimatefix/hexatron/backend/internal/structures/responses"
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
	summaries := c.airflowService.ListServices(
		r.URL.Query().Get("search"),
		r.URL.Query().Get("category"),
	)
	utils.WriteJSON(w, http.StatusOK, toServiceListResponse(summaries))
}

// GetService handles GET /services/{id}.
//
// Response: structures/responses.ServiceDetailResponse (JSON object)
//
// Returns 404 if the service is not found.
func (c *AirflowController) GetService(w http.ResponseWriter, r *http.Request) {
	serviceID := r.PathValue("id")
	detail, found := c.airflowService.GetServiceByID(serviceID)
	if !found {
		utils.WriteError(w, http.StatusNotFound, constants.ErrServiceNotFound+": "+serviceID)
		return
	}
	utils.WriteJSON(w, http.StatusOK, toServiceDetailResponse(detail))
}

// toServiceListResponse maps domain ServiceSummary slice to the HTTP response type.
func toServiceListResponse(summaries []models.ServiceSummary) responses.ServiceListResponse {
	result := make(responses.ServiceListResponse, len(summaries))
	for i, s := range summaries {
		result[i] = responses.ServiceSummary{
			ID:     s.ID,
			Name:   s.Name,
			Status: s.Status,
		}
	}
	return result
}

// toServiceDetailResponse maps a domain ServiceDetail to the HTTP response type.
func toServiceDetailResponse(detail models.ServiceDetail) responses.ServiceDetailResponse {
	dags := make([]responses.DAGStatus, len(detail.DAGs))
	for i, d := range detail.DAGs {
		dags[i] = responses.DAGStatus{
			DAGID:  d.DAGID,
			Status: d.Status,
		}
	}
	return responses.ServiceDetailResponse{
		ID:     detail.ID,
		Name:   detail.Name,
		Status: detail.Status,
		DAGs:   dags,
	}
}
