package controllers

import (
	"net/http"
	"strings"

	"github.com/openclimatefix/hexatron/backend/constants"
	"github.com/openclimatefix/hexatron/backend/services"
	clientstructs "github.com/openclimatefix/hexatron/backend/structures/clients"
	configstructs "github.com/openclimatefix/hexatron/backend/structures/config"
	"github.com/openclimatefix/hexatron/backend/structures/requests"
	"github.com/openclimatefix/hexatron/backend/utils"
)

// AirflowController handles HTTP requests for service and DAG health status.
type AirflowController struct {
	airflowService *services.AirflowService
}

// NewAirflowController constructs an AirflowController with client configuration.
func NewAirflowController(cfg *configstructs.Config) *AirflowController {
	clientCfg := clientstructs.AirflowClientConfig{
		BaseURL: cfg.AirflowBaseURL,
		Cookie:  cfg.AirflowCookie,
	}
	return &AirflowController{
		airflowService: services.NewAirflowService(constants.ServicesConfigPath, clientCfg),
	}
}

// ListServices handles GET /services.
//
// Request:  structures/requests.GetServicesRequestPayload  (query params)
// Response: structures/responses.ServiceListResponse       (JSON array)
//
// Optional query params:
//   - ?search=<string>   case-insensitive name filter
//   - ?category=<string> exact category match
func (c *AirflowController) ListServices(w http.ResponseWriter, r *http.Request) {
	req := requests.GetServicesRequestPayload{
		Search:   r.URL.Query().Get("search"),
		Category: r.URL.Query().Get("category"),
	}

	response := c.airflowService.ListServices(req.Search, req.Category)
	utils.WriteJSON(w, http.StatusOK, response)
}

// GetService handles GET /services/{serviceId}.
//
// Request:  structures/requests.GetServiceDetailRequestPayload (path param)
// Response: structures/responses.ServiceDetailResponse         (JSON object)
//
// Returns 404 if the serviceId is not found.
func (c *AirflowController) GetService(w http.ResponseWriter, r *http.Request) {
	serviceID := strings.TrimPrefix(r.URL.Path, constants.ServiceByIDPath)
	response, found := c.airflowService.GetServiceByID(serviceID)
	if !found {
		utils.WriteError(w, http.StatusNotFound, constants.ErrServiceNotFound+": "+serviceID)
		return
	}

	utils.WriteJSON(w, http.StatusOK, response)
}
