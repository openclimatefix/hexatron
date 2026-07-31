// Package services implements the business logic layer.
package services

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/openclimatefix/hexatron/backend/internal/clients"
	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/models"
	airflowmodels "github.com/openclimatefix/hexatron/backend/internal/models/airflow"
	clientstructs "github.com/openclimatefix/hexatron/backend/internal/structures/clients"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
)

// AirflowService computes service health from Airflow, using data/services.yaml.
type AirflowService struct {
	registry      *ServiceRegistry
	airflowClient *clients.AirflowClient
}

// NewAirflowService creates an AirflowService using application config.
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

// snapshot is one read of Airflow: its DAGs, plus the latest run of the ones a caller wants.
type snapshot struct {
	meta map[string]airflowmodels.DAG
	runs map[string]*airflowmodels.DagRun

	// dagIDs is every dag_id Airflow reported, sorted.
	dagIDs []string
}

// snapshot lists the DAGs, then fetches the latest run of whichever DAGs pick selects.
func (s *AirflowService) snapshot(ctx context.Context, pick func(dagIDs []string) []string) (*snapshot, error) {
	dags, err := s.airflowClient.ListDAGs(ctx)
	if err != nil {
		return nil, err
	}

	meta := make(map[string]airflowmodels.DAG, len(dags))
	dagIDs := make([]string, 0, len(dags))
	for _, dag := range dags {
		meta[dag.DAGID] = dag
		dagIDs = append(dagIDs, dag.DAGID)
	}
	// Airflow's ordering is not part of its contract, so sort for a stable response.
	sort.Strings(dagIDs)

	runs, err := s.airflowClient.GetLatestDagRuns(ctx, pick(dagIDs))
	if err != nil {
		return nil, err
	}

	return &snapshot{meta: meta, runs: runs, dagIDs: dagIDs}, nil
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

// AggregateStatus derives a service status from its DAGs: failed > running > queued > unknown > healthy.
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

// buildDAGStatuses builds the DAG statuses for the given DAG IDs from a snapshot.
func (s *AirflowService) buildDAGStatuses(dagIDs []string, snap *snapshot) []models.DAGStatus {
	statuses := make([]models.DAGStatus, 0, len(dagIDs))

	for _, dagID := range dagIDs {
		status := models.DAGStatus{
			DAGID:      dagID,
			Status:     constants.StatusUnknown,
			AirflowURL: s.airflowClient.DAGURL(dagID),
		}

		// Every dagID came from this snapshot, so its metadata is always present.
		meta := snap.meta[dagID]
		status.Name = meta.Name()
		status.IsPaused = meta.IsPaused
		status.Schedule = meta.Schedule.String()

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

// claimedDAGIDs returns the DAGs at least one service claims, deduplicated.
func (s *AirflowService) claimedDAGIDs(dagIDs []string) []string {
	seen := make(map[string]struct{}, len(dagIDs))
	out := make([]string, 0, len(dagIDs))

	for _, svc := range s.registry.All() {
		for _, dagID := range svc.MatchDAGs(dagIDs) {
			if _, ok := seen[dagID]; ok {
				continue
			}
			seen[dagID] = struct{}{}
			out = append(out, dagID)
		}
	}
	return out
}

// ListServices returns all configured services matching the optional search and category filters.
func (s *AirflowService) ListServices(ctx context.Context, search, category string) ([]models.ServiceSummary, error) {
	snap, err := s.snapshot(ctx, s.claimedDAGIDs)
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

		dagStatuses := s.buildDAGStatuses(service.MatchDAGs(snap.dagIDs), snap)

		result = append(result, models.ServiceSummary{
			ID:       service.ID,
			Name:     service.Name,
			Category: service.Category,
			Status:   AggregateStatus(dagStatuses),
		})
	}
	return result, nil
}

// GetServiceByID returns the detail for a single service, and whether it is configured.
func (s *AirflowService) GetServiceByID(ctx context.Context, serviceID string) (models.ServiceDetail, bool, error) {
	service, found := s.registry.ByID(serviceID)
	if !found {
		return models.ServiceDetail{}, false, nil
	}

	snap, err := s.snapshot(ctx, service.MatchDAGs)
	if err != nil {
		return models.ServiceDetail{}, true, err
	}

	dagStatuses := s.buildDAGStatuses(service.MatchDAGs(snap.dagIDs), snap)

	return models.ServiceDetail{
		ID:       service.ID,
		Name:     service.Name,
		Category: service.Category,
		Status:   AggregateStatus(dagStatuses),
		DAGs:     dagStatuses,
	}, true, nil
}

// ConfigDrift reports patterns claiming no DAG, as "service-id: pattern", and unclaimed DAGs.
func (s *AirflowService) ConfigDrift(ctx context.Context) (unmatched, unclaimed []string, err error) {
	dags, err := s.airflowClient.ListDAGs(ctx)
	if err != nil {
		return nil, nil, err
	}

	dagIDs := make([]string, 0, len(dags))
	for _, dag := range dags {
		dagIDs = append(dagIDs, dag.DAGID)
	}
	sort.Strings(dagIDs)

	claimed := make(map[string]struct{}, len(dagIDs))
	for _, svc := range s.registry.All() {
		for _, pattern := range svc.DAGPatterns {
			matches := models.MatchDAGPattern(pattern, dagIDs)
			if len(matches) == 0 {
				unmatched = append(unmatched, fmt.Sprintf("%s: %s", svc.ID, pattern))
				continue
			}
			for _, dagID := range matches {
				claimed[dagID] = struct{}{}
			}
		}
	}

	for _, dagID := range dagIDs {
		if _, ok := claimed[dagID]; !ok {
			unclaimed = append(unclaimed, dagID)
		}
	}
	return unmatched, unclaimed, nil
}
