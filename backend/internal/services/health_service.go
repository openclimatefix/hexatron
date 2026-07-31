package services

import (
	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/structures/responses"
)

// HealthService is the concrete implementation of the HealthService interface.
type HealthService struct{}

// NewHealthService returns a new HealthService instance.
func NewHealthService() *HealthService {
	return &HealthService{}
}

// GetHealth returns the current operational health of the Hexatron API.
func (s *HealthService) GetHealth() responses.HealthResponse {
	return responses.HealthResponse{Status: constants.StatusHealthy}
}
