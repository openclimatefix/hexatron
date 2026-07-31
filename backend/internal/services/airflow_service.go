// Package services implements the business logic layer.
package services

import (
	"log"
	"strings"

	"github.com/openclimatefix/hexatron/backend/internal/clients"
	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/models"
	clientstructs "github.com/openclimatefix/hexatron/backend/internal/structures/clients"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
)

// AirflowService is the concrete implementation of the AirflowService interface.
// Service definitions are loaded dynamically from data/services.yaml.
type AirflowService struct {
	registry      *ServiceRegistry
	airflowClient *clients.AirflowClient
}

// NewAirflowService creates an AirflowService using application config.
// It maps AirflowBaseURL and AirflowCookie from cfg to the client internally.
func NewAirflowService(cfg *configstructs.Config) *AirflowService {
	registry, err := NewServiceRegistry(cfg.ServicesConfigPath)
	if err != nil {
		log.Fatalf("failed to initialize service registry from %s: %v", cfg.ServicesConfigPath, err)
	}

	airflowClient := clients.NewAirflowClient(clientstructs.AirflowClientConfig{
		BaseURL: cfg.AirflowBaseURL,
		Cookie:  cfg.AirflowCookie,
	})

	return &AirflowService{
		registry:      registry,
		airflowClient: airflowClient,
	}
}

// mapAirflowStateToStatus maps an Airflow DAG state string to a service status.
func mapAirflowStateToStatus(state string) string {
	switch state {
	case constants.AirflowStateSuccess:
		return constants.StatusHealthy
	case constants.AirflowStateFailed:
		return constants.StatusFailed
	default:
		return constants.StatusUnknown
	}
}

// getDAGStatus fetches the runtime status for a given DAG ID.
func (s *AirflowService) getDAGStatus(dagID string) string {
	dagRun, err := s.airflowClient.GetLatestDagRun(dagID)
	if err == nil && dagRun != nil {
		return mapAirflowStateToStatus(dagRun.State)
	}
	return constants.StatusUnknown
}

// aggregateStatus derives the overall service status from its DAG statuses.
// Returns "failed" if any DAG is failed or unknown; otherwise "healthy".
func aggregateStatus(dags []models.DAGStatus) string {
	for _, d := range dags {
		if d.Status == constants.StatusFailed || d.Status == constants.StatusUnknown {
			return constants.StatusFailed
		}
	}
	return constants.StatusHealthy
}

// buildDAGStatuses builds a []models.DAGStatus for the given DAG IDs using the provided status function.
func buildDAGStatuses(dagIDs []string, statusFn func(string) string) []models.DAGStatus {
	statuses := make([]models.DAGStatus, 0, len(dagIDs))
	for _, dagID := range dagIDs {
		statuses = append(statuses, models.DAGStatus{
			DAGID:  dagID,
			Status: statusFn(dagID),
		})
	}
	return statuses
}

// ListServices returns all configured services matching the optional search and category filters.
func (s *AirflowService) ListServices(search, category string) []models.ServiceSummary {
	result := make([]models.ServiceSummary, 0)
	for _, service := range s.registry.All() {
		if search != "" && !strings.Contains(strings.ToLower(service.Name), strings.ToLower(search)) {
			continue
		}
		if category != "" && !strings.EqualFold(service.Category, category) {
			continue
		}

		dagStatuses := buildDAGStatuses(service.DAGIDs, s.getDAGStatus)

		result = append(result, models.ServiceSummary{
			ID:     service.ID,
			Name:   service.Name,
			Status: aggregateStatus(dagStatuses),
		})
	}
	return result
}

// GetServiceByID returns the detail for a single service from services.yaml.
func (s *AirflowService) GetServiceByID(serviceID string) (models.ServiceDetail, bool) {
	service, found := s.registry.ByID(serviceID)
	if !found {
		return models.ServiceDetail{}, false
	}

	dagStatuses := buildDAGStatuses(service.DAGIDs, s.getDAGStatus)

	return models.ServiceDetail{
		ID:     service.ID,
		Name:   service.Name,
		Status: aggregateStatus(dagStatuses),
		DAGs:   dagStatuses,
	}, true
}
