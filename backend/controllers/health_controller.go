package controllers

import (
	"net/http"

	"github.com/openclimatefix/hexatron/backend/services"
	"github.com/openclimatefix/hexatron/backend/utils"
)

var healthSvc = &services.HealthSvc{}

// HealthCheck handles GET /health.
//
// Request:  structures/requests.HealthRequestPayload  (no params)
// Response: structures/responses.HealthResponse       (JSON object)
//
// Returns a 200 with { "status": "healthy" } when the API is operational.
func HealthCheck(w http.ResponseWriter, r *http.Request) {
	response := healthSvc.GetHealth()
	utils.WriteJSON(w, http.StatusOK, response)
}
