// Package controllers handles incoming HTTP requests and delegates to the
// service layer. Each function maps 1:1 to a REST endpoint.
package controllers

import (
	"net/http"
	"strings"

	"github.com/openclimatefix/hexatron/backend/constants"
	"github.com/openclimatefix/hexatron/backend/services"
	"github.com/openclimatefix/hexatron/backend/structures/requests"
	"github.com/openclimatefix/hexatron/backend/utils"
)

var airflowService = services.NewAirflowService(constants.ServicesConfigPath)

// ListServices handles GET /services.
//
// Request:  structures/requests.GetServicesRequestPayload  (query params)
// Response: structures/responses.ServiceListResponse       (JSON array)
//
// Optional query params:
//   - ?search=<string>   case-insensitive name filter
//   - ?category=<string> exact category match
func ListServices(w http.ResponseWriter, r *http.Request) {
	req := requests.GetServicesRequestPayload{
		Search:   r.URL.Query().Get("search"),
		Category: r.URL.Query().Get("category"),
	}

	response := airflowService.ListServices(req.Search, req.Category)
	utils.WriteJSON(w, http.StatusOK, response)
}

// GetService handles GET /services/{serviceId}.
//
// Request:  structures/requests.GetServiceDetailRequestPayload (path param)
// Response: structures/responses.ServiceDetailResponse         (JSON object)
//
// Returns 404 if the serviceId is not found.
func GetService(w http.ResponseWriter, r *http.Request) {
	serviceID := strings.TrimPrefix(r.URL.Path, constants.ServiceByIDPath)
	response, found := airflowService.GetServiceByID(serviceID)
	if !found {
		utils.WriteError(w, http.StatusNotFound, constants.ErrServiceNotFound+": "+serviceID)
		return
	}

	utils.WriteJSON(w, http.StatusOK, response)
}
