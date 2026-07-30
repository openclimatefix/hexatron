// Package services implements the business logic layer.
package services

import (
	"log"
	"strings"

	"github.com/openclimatefix/hexatron/backend/constants"
	"github.com/openclimatefix/hexatron/backend/structures/mock"
	"github.com/openclimatefix/hexatron/backend/structures/responses"
)

// AirflowSvc is the concrete implementation of the AirflowService interface.
// Service definitions are loaded dynamically from data/services.yaml.
type AirflowSvc struct {
	registry *ServiceRegistry
}

// NewAirflowSvc creates an AirflowSvc loaded with services from services.yaml.
func NewAirflowSvc(configPath string) *AirflowSvc {
	registry, err := NewServiceRegistry(configPath)
	if err != nil {
		log.Fatalf("failed to initialize service registry from %s: %v", configPath, err)
	}
	return &AirflowSvc{registry: registry}
}

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

// ListServices returns all configured services loaded from services.yaml.
func (s *AirflowSvc) ListServices(search, category string) responses.ServiceListResponse {
	result := make(responses.ServiceListResponse, 0)
	for _, svc := range s.registry.All() {
		if search != "" && !strings.Contains(strings.ToLower(svc.Name), strings.ToLower(search)) {
			continue
		}
		if category != "" && !strings.EqualFold(svc.Category, category) {
			continue
		}

		dagStatuses := make([]responses.DAGStatus, 0, len(svc.DAGIDs))
		for _, dagID := range svc.DAGIDs {
			dagStatuses = append(dagStatuses, responses.DAGStatus{
				DAGID:  dagID,
				Status: mock.GetDAGStatus(dagID),
			})
		}

		result = append(result, responses.ServiceSummary{
			ID:     svc.ID,
			Name:   svc.Name,
			Status: aggregateStatus(dagStatuses),
		})
	}
	return result
}

// GetServiceByID returns the detail response for a single service from services.yaml.
func (s *AirflowSvc) GetServiceByID(serviceID string) (responses.ServiceDetailResponse, bool) {
	svc, found := s.registry.ByID(serviceID)
	if !found {
		return responses.ServiceDetailResponse{}, false
	}

	dagStatuses := make([]responses.DAGStatus, 0, len(svc.DAGIDs))
	for _, dagID := range svc.DAGIDs {
		dagStatuses = append(dagStatuses, responses.DAGStatus{
			DAGID:  dagID,
			Status: mock.GetDAGStatus(dagID),
		})
	}

	return responses.ServiceDetailResponse{
		ID:     svc.ID,
		Name:   svc.Name,
		Status: aggregateStatus(dagStatuses),
		DAGs:   dagStatuses,
	}, true
}
