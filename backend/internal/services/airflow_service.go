// Package services implements the business logic layer.
package services

import (
	"context"
	"log"
	"strings"

	"github.com/openclimatefix/hexatron/backend/internal/clients"
	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/models"
	airflowmodels "github.com/openclimatefix/hexatron/backend/internal/models/airflow"
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

// snapshot is one consistent read of Airflow: DAG metadata plus the latest run
// of each requested DAG. Building it once per request keeps a response
// internally consistent and bounds the number of Airflow calls.
type snapshot struct {
	meta map[string]airflowmodels.DAG
	runs map[string]*airflowmodels.DagRun
}

func (s *AirflowService) snapshot(ctx context.Context, dagIDs []string) (*snapshot, error) {
	dags, err := s.airflowClient.ListDAGs(ctx)
	if err != nil {
		return nil, err
	}

	meta := make(map[string]airflowmodels.DAG, len(dags))
	for _, dag := range dags {
		meta[dag.DAGID] = dag
	}

	runs, err := s.airflowClient.GetLatestDagRuns(ctx, dagIDs)
	if err != nil {
		return nil, err
	}

	return &snapshot{meta: meta, runs: runs}, nil
}

// MapAirflowStateToStatus maps an Airflow DAG state string to a service status.
func MapAirflowStateToStatus(state string) string {
	switch state {
	case constants.AirflowStateSuccess:
		return constants.StatusHealthy
	case constants.AirflowStateFailed, constants.AirflowStateUpstreamFailed:
		return constants.StatusFailed
	case constants.AirflowStateRunning:
		return constants.StatusRunning
	case constants.AirflowStateQueued:
		return constants.StatusQueued
	default:
		return constants.StatusUnknown
	}
}

// AggregateStatus derives the overall service status from its DAG statuses.
// Failure outranks everything so a broken DAG is never masked by a busy one:
//
//	any failed  -> failed
//	any running -> running
//	any queued  -> queued
//	any unknown -> unknown
//	otherwise   -> healthy
func AggregateStatus(dags []models.DAGStatus) string {
	if len(dags) == 0 {
		return constants.StatusUnknown
	}

	var running, queued, unknown bool
	for _, d := range dags {
		switch d.Status {
		case constants.StatusFailed:
			return constants.StatusFailed
		case constants.StatusRunning:
			running = true
		case constants.StatusQueued:
			queued = true
		case constants.StatusHealthy:
		default:
			unknown = true
		}
	}

	switch {
	case running:
		return constants.StatusRunning
	case queued:
		return constants.StatusQueued
	case unknown:
		return constants.StatusUnknown
	default:
		return constants.StatusHealthy
	}
}

// buildDAGStatuses builds a []models.DAGStatus for the given DAG IDs from a
// snapshot of Airflow.
func (s *AirflowService) buildDAGStatuses(dagIDs []string, snap *snapshot) []models.DAGStatus {
	statuses := make([]models.DAGStatus, 0, len(dagIDs))

	for _, dagID := range dagIDs {
		status := models.DAGStatus{
			DAGID:      dagID,
			Status:     constants.StatusUnknown,
			AirflowURL: s.airflowClient.DAGURL(dagID),
		}

		// A DAG in services.yaml that Airflow has never heard of stays unknown.
		if meta, ok := snap.meta[dagID]; ok {
			status.Name = meta.Name()
			status.IsPaused = meta.IsPaused
			status.Schedule = meta.Schedule.String()
		}

		if run := snap.runs[dagID]; run != nil {
			status.Status = MapAirflowStateToStatus(run.State)
			status.LastRun = &models.RunSummary{
				RunID:     run.DagRunID,
				State:     run.State,
				StartDate: run.StartDate,
				EndDate:   run.EndDate,
			}
		}

		statuses = append(statuses, status)
	}
	return statuses
}

// allDAGIDs returns every DAG referenced by any configured service,
// deduplicated. Services may legitimately share a DAG.
func (s *AirflowService) allDAGIDs() []string {
	seen := make(map[string]struct{})
	var out []string

	for _, svc := range s.registry.All() {
		for _, dagID := range svc.DAGIDs {
			if _, ok := seen[dagID]; ok {
				continue
			}
			seen[dagID] = struct{}{}
			out = append(out, dagID)
		}
	}
	return out
}

// ListServices returns all configured services matching the optional search and
// category filters. The DAG breakdown is omitted; use GetServiceByID for that.
func (s *AirflowService) ListServices(ctx context.Context, search, category string) ([]models.ServiceSummary, error) {
	snap, err := s.snapshot(ctx, s.allDAGIDs())
	if err != nil {
		return nil, err
	}

	result := make([]models.ServiceSummary, 0)
	for _, service := range s.registry.All() {
		if search != "" && !strings.Contains(strings.ToLower(service.Name), strings.ToLower(search)) {
			continue
		}
		if category != "" && !strings.EqualFold(service.Category, category) {
			continue
		}

		dagStatuses := s.buildDAGStatuses(service.DAGIDs, snap)

		result = append(result, models.ServiceSummary{
			ID:       service.ID,
			Name:     service.Name,
			Category: service.Category,
			Status:   AggregateStatus(dagStatuses),
		})
	}
	return result, nil
}

// GetServiceByID returns the detail for a single service from services.yaml.
// The boolean reports whether the service is configured at all, which the
// controller turns into a 404.
func (s *AirflowService) GetServiceByID(ctx context.Context, serviceID string) (models.ServiceDetail, bool, error) {
	service, found := s.registry.ByID(serviceID)
	if !found {
		return models.ServiceDetail{}, false, nil
	}

	snap, err := s.snapshot(ctx, service.DAGIDs)
	if err != nil {
		return models.ServiceDetail{}, true, err
	}

	dagStatuses := s.buildDAGStatuses(service.DAGIDs, snap)

	return models.ServiceDetail{
		ID:       service.ID,
		Name:     service.Name,
		Category: service.Category,
		Status:   AggregateStatus(dagStatuses),
		DAGs:     dagStatuses,
	}, true, nil
}

// ConfigDrift reports mismatches between services.yaml and Airflow: DAGs that
// are configured but missing from Airflow, and DAGs Airflow runs that no service
// claims. Both are silent gaps in a dashboard, so they are worth surfacing.
func (s *AirflowService) ConfigDrift(ctx context.Context) (missing, unclaimed []string, err error) {
	dags, err := s.airflowClient.ListDAGs(ctx)
	if err != nil {
		return nil, nil, err
	}

	inAirflow := make(map[string]struct{}, len(dags))
	for _, dag := range dags {
		inAirflow[dag.DAGID] = struct{}{}
	}

	configured := make(map[string]struct{})
	for _, dagID := range s.allDAGIDs() {
		configured[dagID] = struct{}{}
		if _, ok := inAirflow[dagID]; !ok {
			missing = append(missing, dagID)
		}
	}

	for _, dag := range dags {
		if _, ok := configured[dag.DAGID]; !ok {
			unclaimed = append(unclaimed, dag.DAGID)
		}
	}
	return missing, unclaimed, nil
}
