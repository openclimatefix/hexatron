// Package interfaces defines the service layer contracts.
package interfaces

import "github.com/openclimatefix/hexatron/backend/structures/responses"

// AirflowService defines the contract for the business service layer that
// reads service health from Airflow DAG runs.
type AirflowService interface {
	// ListServices returns all services filtered by optional search and category.
	ListServices(search, category string) responses.ServiceListResponse

	// GetServiceByID returns the detail for a single service and true,
	// or an empty value and false if the service ID is not found.
	GetServiceByID(serviceID string) (responses.ServiceDetailResponse, bool)
}
