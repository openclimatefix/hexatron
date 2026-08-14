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

// AirflowService computes service health from Airflow, using data/services.yaml.
type AirflowService struct {
	registry      *ServiceRegistry
	airflowClient *clients.AirflowClient
	heartbeats    *HeartbeatService
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
		heartbeats:    NewHeartbeatService(),
	}
}

// fullSnapshot is one read of Airflow: its DAGs, plus the recent runs of each.
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

	// Probed once for the whole list rather than per service, so the concurrent
	// fan-out happens regardless of how far the filters narrow the result.
	beats := s.heartbeats.CheckAll(ctx, allSvcs)

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

		var hb *responses.Heartbeat
		if beat, ok := beats[svc.ID]; ok {
			hb = &beat
		}

		svcResp := s.buildServiceResponse(svc, dagIDs, snap, hb)
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
	svcResp := s.buildServiceResponse(svc, dagIDs, snap, s.heartbeats.Check(ctx, svc))

	return svcResp, true, nil
}

func (s *AirflowService) buildServiceResponse(svc models.Service, dagIDs []string, snap *fullSnapshot, hb *responses.Heartbeat) responses.ServiceResponse {
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
	dagStatus, statusReason := deriveStatus(svc, dags, metrics)
	status := ApplyHeartbeat(dagStatus, hb)

	// A heartbeat that overrode the DAG verdict owns the explanation too —
	// otherwise a card reads "Down" above a reason about a DAG that is fine.
	if status != dagStatus && hb != nil {
		statusReason = heartbeatReason(hb)
	}

	return responses.ServiceResponse{
		ID:           svc.ID,
		Name:         svc.Name,
		Category:     svc.Category,
		Status:       status,
		StatusReason: statusReason,
		DependsOn:    svc.DependsOn,
		DAGs:         dags,
		Metrics:      metrics,
		Note:         nil,
		Heartbeat:    hb,
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

	for i, run := range respRuns {
		totalRuns++
		if run.State == constants.AirflowStateFailed || run.State == constants.AirflowStateUpstreamFailed {
			failedRuns++
		}

		if run.StartDate != nil && run.EndDate != nil {
			start, err1 := time.Parse(time.RFC3339, *run.StartDate)
			end, err2 := time.Parse(time.RFC3339, *run.EndDate)
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
		} else if run.StartDate != nil && lastRunAt == nil {
			start, err := time.Parse(time.RFC3339, *run.StartDate)
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

	for _, dag := range dags {
		meta, exists := dagMeta[dag.DAGID]
		if exists && meta.NextDagRun != nil {
			if nextRunTime.IsZero() || meta.NextDagRun.Before(nextRunTime) {
				nextRunTime = *meta.NextDagRun
				str := meta.NextDagRun.Format(time.RFC3339)
				nextRunAt = &str
			}
		}

		for _, run := range dag.Runs {
			totalRuns++
			if run.State == constants.AirflowStateFailed || run.State == constants.AirflowStateUpstreamFailed {
				failedRuns++
			}

			if run.StartDate != nil && run.EndDate != nil {
				start, err1 := time.Parse(time.RFC3339, *run.StartDate)
				end, err2 := time.Parse(time.RFC3339, *run.EndDate)
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
			} else if run.StartDate != nil {
				start, err := time.Parse(time.RFC3339, *run.StartDate)
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

// deriveStatus computes a service's status and, when something is wrong, a
// short phrase naming what.
//
// The reason matters as much as the status: a red card that does not say which
// DAG failed sends the operator hunting through a modal to find out.
func deriveStatus(svc models.Service, dags []responses.DAGDetail, metrics responses.ServiceMetrics) (string, *string) {
	if svc.Planned && len(dags) == 0 {
		return constants.StatusPlanned, nil
	}

	if len(dags) == 0 {
		return constants.StatusUnknown, nil
	}

	allPaused := true
	for _, dag := range dags {
		if !dag.IsPaused {
			allPaused = false
			break
		}
	}
	if allPaused {
		return constants.StatusPaused, nil
	}

	if metrics.TotalRuns == 0 || metrics.SuccessRate == nil {
		return constants.StatusUnknown, nil
	}

	// Collected rather than short-circuited: naming the failing DAGs is the
	// point, and which of them are critical decides the severity.
	var criticalFailures, nonCriticalFailures []string
	hasRunning := false

	for _, dag := range dags {
		if len(dag.Runs) == 0 {
			continue
		}
		switch dag.Runs[0].State {
		case constants.AirflowStateFailed, constants.AirflowStateUpstreamFailed:
			if svc.IsCritical(dag.DAGID) {
				criticalFailures = append(criticalFailures, dag.DAGID)
			} else {
				nonCriticalFailures = append(nonCriticalFailures, dag.DAGID)
			}
		case constants.AirflowStateRunning:
			hasRunning = true
		}
	}

	if len(criticalFailures) > 0 {
		return constants.StatusFailed, reason(criticalFailures, "failing")
	}

	// A non-critical DAG cannot down the service, but it must not be silent
	// either — degraded, and named.
	if len(nonCriticalFailures) > 0 {
		return constants.StatusDegraded, reason(nonCriticalFailures, "failing (non-critical)")
	}

	if hasRunning {
		return constants.StatusRunning, nil
	}

	if *metrics.SuccessRate < 0.95 {
		pct := *metrics.SuccessRate * 100
		msg := fmt.Sprintf("%.1f%% success over %d runs", pct, metrics.TotalRuns)
		return constants.StatusDegraded, &msg
	}

	return constants.StatusHealthy, nil
}

// reason names the offending DAGs, summarising past two so the phrase stays
// short enough to sit on a card.
func reason(dagIDs []string, suffix string) *string {
	var subject string
	switch len(dagIDs) {
	case 1:
		subject = dagIDs[0]
	case 2:
		subject = dagIDs[0] + " and " + dagIDs[1]
	default:
		subject = fmt.Sprintf("%s and %d others", dagIDs[0], len(dagIDs)-1)
	}
	msg := subject + " " + suffix
	return &msg
}

func mapStateToStatus(state string) string {
	return MapAirflowStateToStatus(state)
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

// AggregateStatus derives a service status from its DAGs: failed > running > queued > unknown > healthy.
func AggregateStatus(dags []models.DAGStatus) string {
	if len(dags) == 0 {
		return constants.StatusUnknown
	}

	var running, queued, unknown bool
	for _, dag := range dags {
		switch dag.Status {
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
