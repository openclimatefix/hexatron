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

// snapshot is one consistent read of Airflow: every DAG it reports, plus the
// latest run of those a caller cares about. Building it once per request keeps
// a response internally consistent and bounds the number of Airflow calls.
type snapshot struct {
	meta map[string]airflowmodels.DAG
	runs map[string]*airflowmodels.DagRun

	// dagIDs is every dag_id Airflow reported, sorted. Service membership is
	// resolved against this rather than a list in services.yaml, so Airflow is
	// the only place DAGs are named.
	dagIDs []string
}

// snapshot reads the DAG list first, then the latest run of whichever DAGs pick
// selects from it. Runs are the expensive half — one call per DAG — so pick
// exists to keep a single-service request from fetching the whole deployment.
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
	// Airflow's ordering is not part of its contract; sorting keeps the DAGs in
	// a response stable from one request to the next.
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

		// Every dagID here was matched against this same snapshot, so the metadata
		// is always present.
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

// claimedDAGIDs returns the DAGs at least one service claims, deduplicated,
// given every dag id Airflow reported. Services may legitimately share a DAG,
// and a DAG no service claims is never fetched.
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

// ListServices returns all configured services matching the optional search and
// category filters. The DAG breakdown is omitted; use GetServiceByID for that.
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

// GetServiceByID returns the detail for a single service from services.yaml.
// The boolean reports whether the service is configured at all, which the
// controller turns into a 404.
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

// ConfigDrift reports mismatches between services.yaml and Airflow: patterns
// that claim no DAG at all, reported as "service-id: pattern", and DAGs Airflow
// runs that no service claims. Both are silent gaps in a dashboard, so they are
// worth surfacing.
//
// A pattern claiming nothing is the equivalent of the typo'd dag id it replaced
// — a renamed DAG leaves its service quietly empty. A service with no patterns
// at all is not drift: wind-forecast and ui are waiting on DAGs that do not
// exist yet.
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
