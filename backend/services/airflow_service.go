// Package services implements the business logic layer.
package services

import (
	"log"
	"strings"

	"github.com/openclimatefix/hexatron/backend/constants"
	"github.com/openclimatefix/hexatron/backend/structures/mock"
	"github.com/openclimatefix/hexatron/backend/structures/responses"
)

// AirflowService is the concrete implementation of the AirflowService interface.
// Service definitions are loaded dynamically from data/services.yaml.
type AirflowService struct {
	registry *ServiceRegistry
}

// NewAirflowService creates an AirflowService loaded with services from services.yaml.
func NewAirflowService(configPath string) *AirflowService {
	registry, err := NewServiceRegistry(configPath)
	if err != nil {
		log.Fatalf("failed to initialize service registry from %s: %v", configPath, err)
	}
	return &AirflowService{registry: registry}
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
func (s *AirflowService) ListServices(search, category string) responses.ServiceListResponse {
	result := make(responses.ServiceListResponse, 0)
	for _, service := range s.registry.All() {
		if search != "" && !strings.Contains(strings.ToLower(service.Name), strings.ToLower(search)) {
			continue
		}
		if category != "" && !strings.EqualFold(service.Category, category) {
			continue
		}

		dagStatuses := make([]responses.DAGStatus, 0, len(service.DAGIDs))
		for _, dagID := range service.DAGIDs {
			dagStatuses = append(dagStatuses, responses.DAGStatus{
				DAGID:  dagID,
				Status: mock.GetDAGStatus(dagID),
			})
		}

		result = append(result, responses.ServiceSummary{
			ID:     service.ID,
			Name:   service.Name,
			Status: aggregateStatus(dagStatuses),
		})
	}
	return result
}

// GetServiceByID returns the detail response for a single service from services.yaml.
func (s *AirflowService) GetServiceByID(serviceID string) (responses.ServiceDetailResponse, bool) {
	service, found := s.registry.ByID(serviceID)
	if !found {
		return responses.ServiceDetailResponse{}, false
	}

	dagStatuses := make([]responses.DAGStatus, 0, len(service.DAGIDs))
	for _, dagID := range service.DAGIDs {
		dagStatuses = append(dagStatuses, responses.DAGStatus{
			DAGID:  dagID,
			Status: mock.GetDAGStatus(dagID),
		})
	}

	return responses.ServiceDetailResponse{
		ID:     service.ID,
		Name:   service.Name,
		Status: aggregateStatus(dagStatuses),
		DAGs:   dagStatuses,
	}, true
}
