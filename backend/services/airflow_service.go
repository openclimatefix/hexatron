// Package services implements the business logic layer.
// In Phase 2, these services will delegate to the Airflow client.
package services

import (
	"strings"

	"github.com/openclimatefix/hexatron/backend/constants"
	"github.com/openclimatefix/hexatron/backend/structures/mock"
	"github.com/openclimatefix/hexatron/backend/structures/responses"
)

// AirflowSvc is the concrete implementation of the AirflowService interface.
type AirflowSvc struct{}

// aggregateStatus derives the overall service status from its DAG statuses.
// Returns "failed" if any DAG is failed or unknown; otherwise "healthy".
func aggregateStatus(dags []responses.DAGStatus) string {
	for _, d := range dags {
		if d.Status == constants.StatusFailed || d.Status == constants.StatusUnknown {
			return constants.StatusFailed
		}
	}
	return constants.StatusHealthy
}

// ListServices returns all services, optionally filtered by name search and category.
func (s *AirflowSvc) ListServices(search, category string) responses.ServiceListResponse {
	result := make(responses.ServiceListResponse, 0)
	for _, svc := range mock.MockServiceStore {
		if search != "" && !strings.Contains(strings.ToLower(svc.Name), strings.ToLower(search)) {
			continue
		}
		if category != "" && !strings.EqualFold(svc.Category, category) {
			continue
		}
		result = append(result, responses.ServiceSummary{
			ID:     svc.ID,
			Name:   svc.Name,
			Status: aggregateStatus(svc.DAGs),
		})
	}
	return result
}

// GetServiceByID returns the detail response for a single service.
// Returns false if no service with the given ID exists.
func (s *AirflowSvc) GetServiceByID(serviceID string) (responses.ServiceDetailResponse, bool) {
	for _, svc := range mock.MockServiceStore {
		if svc.ID != serviceID {
			continue
		}
		return responses.ServiceDetailResponse{
			ID:     svc.ID,
			Name:   svc.Name,
			Status: aggregateStatus(svc.DAGs),
			DAGs:   svc.DAGs,
		}, true
	}
	return responses.ServiceDetailResponse{}, false
}
