// Package services implements the business logic layer.
package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	"github.com/openclimatefix/hexatron/backend/internal/clients"
	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/models"
	airflowmodels "github.com/openclimatefix/hexatron/backend/internal/models/airflow"
	clientstructs "github.com/openclimatefix/hexatron/backend/internal/structures/clients"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
	"github.com/openclimatefix/hexatron/backend/internal/structures/responses"
)

// AirflowService is the concrete implementation of the AirflowService interface.
// Service definitions are loaded dynamically from data/services.yaml.
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

type fullSnapshot struct {
	meta map[string]airflowmodels.DAG
	runs map[string][]airflowmodels.DagRun
}

// ListServices returns all configured services matching optional search and category filters.
func (s *AirflowService) ListServices(ctx context.Context, search, category string) (responses.ServiceListResponse, error) {
	allSvcs := s.registry.All()

	dags, err := s.airflowClient.ListDAGs(ctx)
	if err != nil {
		return nil, err
	}

	meta := make(map[string]airflowmodels.DAG, len(dags))
	allDAGIDs := make([]string, 0, len(dags))
	for _, d := range dags {
		meta[d.DAGID] = d
		allDAGIDs = append(allDAGIDs, d.DAGID)
	}

	claimedIDs := s.claimedDAGIDs(allDAGIDs)
	runsMap, err := s.airflowClient.GetRecentDagRunsForDAGs(ctx, claimedIDs, 10)
	if err != nil && !errors.Is(err, clients.ErrDAGNotFound) {
		return nil, err
	}
	if runsMap == nil {
		runsMap = make(map[string][]airflowmodels.DagRun)
	}

	snap := &fullSnapshot{meta: meta, runs: runsMap}

	searchLower := strings.ToLower(strings.TrimSpace(search))
	categoryLower := strings.ToLower(strings.TrimSpace(category))

	result := make(responses.ServiceListResponse, 0, len(allSvcs))
	for _, svc := range allSvcs {
		if categoryLower != "" && strings.ToLower(svc.Category) != categoryLower {
			continue
		}

		dagIDs := svc.MatchDAGs(allDAGIDs)

		if searchLower != "" {
			matchName := strings.Contains(strings.ToLower(svc.Name), searchLower)
			matchCategory := strings.Contains(strings.ToLower(svc.Category), searchLower)
			matchDAG := false
			for _, id := range dagIDs {
				if strings.Contains(strings.ToLower(id), searchLower) {
					matchDAG = true
					break
				}
			}
			if !matchName && !matchCategory && !matchDAG {
				continue
			}
		}

		svcResp := s.buildServiceResponse(svc, dagIDs, snap)
		result = append(result, svcResp)
	}

	return result, nil
}

// GetServiceByID returns the detail for a single service from services.yaml.
func (s *AirflowService) GetServiceByID(ctx context.Context, serviceID string) (responses.ServiceResponse, bool, error) {
	svc, found := s.registry.ByID(serviceID)
	if !found {
		return responses.ServiceResponse{}, false, nil
	}

	dags, err := s.airflowClient.ListDAGs(ctx)
	if err != nil {
		return responses.ServiceResponse{}, true, err
	}

	meta := make(map[string]airflowmodels.DAG, len(dags))
	allDAGIDs := make([]string, 0, len(dags))
	for _, d := range dags {
		meta[d.DAGID] = d
		allDAGIDs = append(allDAGIDs, d.DAGID)
	}

	dagIDs := svc.MatchDAGs(allDAGIDs)
	runsMap, err := s.airflowClient.GetRecentDagRunsForDAGs(ctx, dagIDs, 10)
	if err != nil && !errors.Is(err, clients.ErrDAGNotFound) {
		return responses.ServiceResponse{}, true, err
	}
	if runsMap == nil {
		runsMap = make(map[string][]airflowmodels.DagRun)
	}

	snap := &fullSnapshot{meta: meta, runs: runsMap}
	svcResp := s.buildServiceResponse(svc, dagIDs, snap)

	return svcResp, true, nil
}

func (s *AirflowService) buildServiceResponse(svc models.Service, dagIDs []string, snap *fullSnapshot) responses.ServiceResponse {
	dags := make([]responses.DAGDetail, 0, len(dagIDs))

	for _, id := range dagIDs {
		meta, exists := snap.meta[id]
		rawRuns := snap.runs[id]

		respRuns := make([]responses.DagRun, 0, len(rawRuns))
		for _, r := range rawRuns {
			respRuns = append(respRuns, responses.DagRun{
				DagRunID:    r.DagRunID,
				DAGID:       r.DAGID,
				LogicalDate: formatTimePtr(r.LogicalDate),
				StartDate:   formatTimePtrNull(r.StartDate),
				EndDate:     formatTimePtrNull(r.EndDate),
				State:       r.State,
				RunType:     r.RunType,
			})
		}

		var timetableSummary *string
		if exists && meta.TimetableDescription != "" {
			timetableSummary = &meta.TimetableDescription
		} else if exists && meta.Schedule != nil {
			str := meta.Schedule.String()
			if str != "" {
				timetableSummary = &str
			}
		}

		var nextDagrun *string
		if exists && meta.NextDagRun != nil {
			nextDagrun = formatTimePtrNull(meta.NextDagRun)
		}

		dagDisplayName := id
		isPaused := false
		if exists {
			dagDisplayName = meta.Name()
			isPaused = meta.IsPaused
		}

		dagStatus := "unknown"
		if len(respRuns) > 0 {
			dagStatus = mapStateToStatus(respRuns[0].State)
		} else if isPaused {
			dagStatus = "paused"
		}

		var lastRun *responses.RunSummary
		if len(rawRuns) > 0 {
			lastRun = &responses.RunSummary{
				RunID:     rawRuns[0].DagRunID,
				State:     rawRuns[0].State,
				StartDate: rawRuns[0].StartDate,
				EndDate:   rawRuns[0].EndDate,
			}
		}

		var schedStr string
		if timetableSummary != nil {
			schedStr = *timetableSummary
		}

		var metaPtr *airflowmodels.DAG
		if exists {
			metaPtr = &meta
		}
		dagMetrics := computeDAGMetrics(respRuns, metaPtr)

		dags = append(dags, responses.DAGDetail{
			DAGID:                 id,
			DAGDisplayName:        dagDisplayName,
			IsPaused:              isPaused,
			TimetableSummary:      timetableSummary,
			NextDagrunLogicalDate: nextDagrun,
			Status:                dagStatus,
			AirflowURL:            s.airflowClient.DAGURL(id),
			Schedule:              schedStr,
			LastRun:               lastRun,
			Metrics:               dagMetrics,
			Runs:                  respRuns,
		})
	}

	metrics := computeServiceMetrics(dags, snap.meta)
	status := deriveStatus(dags, metrics)

	return responses.ServiceResponse{
		ID:        svc.ID,
		Name:      svc.Name,
		Category:  svc.Category,
		Status:    status,
		DependsOn: svc.DependsOn,
		DAGs:      dags,
		Metrics:   metrics,
		Note:      nil,
	}
}

func computeDAGMetrics(respRuns []responses.DagRun, meta *airflowmodels.DAG) responses.ServiceMetrics {
	var totalRuns, failedRuns int
	var durationSum float64
	var finishedCount int
	var latestDuration *float64
	var lastRunAt *string
	var nextRunAt *string

	if meta != nil && meta.NextDagRun != nil {
		str := meta.NextDagRun.Format(time.RFC3339)
		nextRunAt = &str
	}

	for i, r := range respRuns {
		totalRuns++
		if r.State == "failed" || r.State == "upstream_failed" {
			failedRuns++
		}

		if r.StartDate != nil && r.EndDate != nil {
			start, err1 := time.Parse(time.RFC3339, *r.StartDate)
			end, err2 := time.Parse(time.RFC3339, *r.EndDate)
			if err1 == nil && err2 == nil && !end.Before(start) {
				dur := end.Sub(start).Seconds()
				durationSum += dur
				finishedCount++

				if i == 0 {
					latestDuration = &dur
				}
				if lastRunAt == nil {
					str := start.Format(time.RFC3339)
					lastRunAt = &str
				}
			}
		} else if r.StartDate != nil && lastRunAt == nil {
			start, err := time.Parse(time.RFC3339, *r.StartDate)
			if err == nil {
				str := start.Format(time.RFC3339)
				lastRunAt = &str
			}
		}
	}

	var successRate *float64
	if totalRuns > 0 {
		sr := float64(totalRuns-failedRuns) / float64(totalRuns)
		successRate = &sr
	}

	var avgDuration *float64
	if finishedCount > 0 {
		avg := durationSum / float64(finishedCount)
		avgDuration = &avg
	}

	return responses.ServiceMetrics{
		TotalRuns:             totalRuns,
		FailedRuns:            failedRuns,
		SuccessRate:           successRate,
		AvgDurationSeconds:    avgDuration,
		LatestDurationSeconds: latestDuration,
		LastRunAt:             lastRunAt,
		NextRunAt:             nextRunAt,
	}
}

func computeServiceMetrics(dags []responses.DAGDetail, dagMeta map[string]airflowmodels.DAG) responses.ServiceMetrics {
	var totalRuns, failedRuns int
	var durationSum float64
	var finishedCount int
	var latestDuration *float64
	var lastRunAt *string
	var nextRunAt *string

	var latestRunTime time.Time
	var nextRunTime time.Time

	for _, d := range dags {
		meta, exists := dagMeta[d.DAGID]
		if exists && meta.NextDagRun != nil {
			if nextRunTime.IsZero() || meta.NextDagRun.Before(nextRunTime) {
				nextRunTime = *meta.NextDagRun
				str := meta.NextDagRun.Format(time.RFC3339)
				nextRunAt = &str
			}
		}

		for _, r := range d.Runs {
			totalRuns++
			if r.State == "failed" || r.State == "upstream_failed" {
				failedRuns++
			}

			if r.StartDate != nil && r.EndDate != nil {
				start, err1 := time.Parse(time.RFC3339, *r.StartDate)
				end, err2 := time.Parse(time.RFC3339, *r.EndDate)
				if err1 == nil && err2 == nil && !end.Before(start) {
					dur := end.Sub(start).Seconds()
					durationSum += dur
					finishedCount++

					if start.After(latestRunTime) {
						latestRunTime = start
						latestDuration = &dur
						str := start.Format(time.RFC3339)
						lastRunAt = &str
					}
				}
			} else if r.StartDate != nil {
				start, err := time.Parse(time.RFC3339, *r.StartDate)
				if err == nil && start.After(latestRunTime) {
					latestRunTime = start
					str := start.Format(time.RFC3339)
					lastRunAt = &str
				}
			}
		}
	}

	var successRate *float64
	if totalRuns > 0 {
		sr := float64(totalRuns-failedRuns) / float64(totalRuns)
		successRate = &sr
	}

	var avgDuration *float64
	if finishedCount > 0 {
		avg := durationSum / float64(finishedCount)
		avgDuration = &avg
	}

	return responses.ServiceMetrics{
		TotalRuns:             totalRuns,
		FailedRuns:            failedRuns,
		SuccessRate:           successRate,
		AvgDurationSeconds:    avgDuration,
		LatestDurationSeconds: latestDuration,
		LastRunAt:             lastRunAt,
		NextRunAt:             nextRunAt,
	}
}

func deriveStatus(dags []responses.DAGDetail, metrics responses.ServiceMetrics) string {
	if len(dags) == 0 {
		return constants.StatusUnknown
	}

	allPaused := true
	for _, d := range dags {
		if !d.IsPaused {
			allPaused = false
			break
		}
	}
	if allPaused {
		return constants.StatusPaused
	}

	if metrics.TotalRuns == 0 || metrics.SuccessRate == nil {
		return constants.StatusUnknown
	}

	hasDown := false
	hasRunning := false
	for _, d := range dags {
		if len(d.Runs) > 0 {
			firstState := d.Runs[0].State
			if firstState == "failed" || firstState == "upstream_failed" {
				hasDown = true
				break
			}
			if firstState == "running" {
				hasRunning = true
			}
		}
	}
	if hasDown {
		return constants.StatusFailed
	}
	if hasRunning {
		return constants.StatusRunning
	}

	if *metrics.SuccessRate < 0.95 {
		return "degraded"
	}

	return constants.StatusHealthy
}

func mapStateToStatus(state string) string {
	switch state {
	case "success":
		return constants.StatusHealthy
	case "failed", "upstream_failed":
		return constants.StatusFailed
	case "running":
		return constants.StatusRunning
	case "queued":
		return constants.StatusQueued
	default:
		return constants.StatusUnknown
	}
}

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

func (s *AirflowService) claimedDAGIDs(allDAGIDs []string) []string {
	seen := make(map[string]struct{})
	out := make([]string, 0)
	for _, svc := range s.registry.All() {
		for _, dagID := range svc.MatchDAGs(allDAGIDs) {
			if _, ok := seen[dagID]; !ok {
				seen[dagID] = struct{}{}
				out = append(out, dagID)
			}
		}
	}
	return out
}

func formatTimePtr(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(time.RFC3339)
}

func formatTimePtrNull(t *time.Time) *string {
	if t == nil {
		return nil
	}
	str := t.Format(time.RFC3339)
	return &str
}
