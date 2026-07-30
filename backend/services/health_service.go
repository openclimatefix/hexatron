package services

import (
	"github.com/openclimatefix/hexatron/backend/constants"
	"github.com/openclimatefix/hexatron/backend/structures/responses"
)

// HealthSvc is the concrete implementation of the HealthService interface.
type HealthSvc struct{}

// GetHealth returns the current operational health of the Hexatron API.
func (s *HealthSvc) GetHealth() responses.HealthResponse {
	return responses.HealthResponse{Status: constants.StatusHealthy}
}
