package interfaces

import "github.com/openclimatefix/hexatron/backend/structures/responses"

// HealthService defines the contract for the health check service.
type HealthService interface {
	// GetHealth returns the current operational health of the Hexatron API.
	GetHealth() responses.HealthResponse
}
