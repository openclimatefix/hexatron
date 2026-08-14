package services

import (
	"context"
	"sync"
	"time"

	"github.com/openclimatefix/hexatron/backend/internal/clients"
	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/models"
	"github.com/openclimatefix/hexatron/backend/internal/structures/responses"
)

// heartbeatTTL bounds how stale a cached probe may be.
//
// The dashboard revalidates every 15s and several clients may watch at once;
// without a cache each viewer would add load to the very services being
// checked, and a heartbeat that degrades its target is worse than none.
const heartbeatTTL = 10 * time.Second

// HeartbeatService probes non-Airflow services and caches the results.
//
// Safe for concurrent use: /services probes every configured service at once.
type HeartbeatService struct {
	client *clients.HeartbeatClient

	mu     sync.Mutex
	cached map[string]cachedHeartbeat
}

type cachedHeartbeat struct {
	result responses.Heartbeat
	at     time.Time
}

// NewHeartbeatService creates a HeartbeatService.
func NewHeartbeatService() *HeartbeatService {
	return &HeartbeatService{
		client: clients.NewHeartbeatClient(),
		cached: make(map[string]cachedHeartbeat),
	}
}

// CheckAll probes every service that configures a health_check, concurrently,
// and returns results keyed by service id. Services without a check are absent
// from the map.
//
// Probes run in parallel because they are almost entirely network wait: run
// serially, a fleet of slow-but-healthy services would add its timeouts
// together and blow the request budget on its own.
func (s *HeartbeatService) CheckAll(ctx context.Context, svcs []models.Service) map[string]responses.Heartbeat {
	results := make(map[string]responses.Heartbeat, len(svcs))
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, svc := range svcs {
		if svc.HealthCheck == nil {
			continue
		}
		wg.Add(1)
		go func(svc models.Service) {
			defer wg.Done()
			hb := s.Check(ctx, svc)
			if hb == nil {
				return
			}
			mu.Lock()
			results[svc.ID] = *hb
			mu.Unlock()
		}(svc)
	}

	wg.Wait()
	return results
}

// Check probes one service, returning a cached result if it is still fresh.
// Returns nil when the service configures no health_check.
func (s *HeartbeatService) Check(ctx context.Context, svc models.Service) *responses.Heartbeat {
	if svc.HealthCheck == nil {
		return nil
	}
	check := svc.HealthCheck.Normalised()

	if hit, ok := s.fresh(svc.ID); ok {
		return &hit
	}

	probe := s.client.Probe(ctx, check)

	hb := responses.Heartbeat{
		Type:      check.Type,
		Target:    check.Target,
		Status:    probeStatus(probe),
		CheckedAt: time.Now().UTC().Format(time.RFC3339),
	}
	// A latency reading only means something when a round trip finished.
	if probe.Serving {
		ms := probe.Latency.Milliseconds()
		hb.LatencyMs = &ms
	}
	if probe.Detail != "" {
		detail := probe.Detail
		hb.Detail = &detail
	}

	s.mu.Lock()
	s.cached[svc.ID] = cachedHeartbeat{result: hb, at: time.Now()}
	s.mu.Unlock()

	return &hb
}

func (s *HeartbeatService) fresh(serviceID string) (responses.Heartbeat, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry, ok := s.cached[serviceID]
	if !ok || time.Since(entry.at) > heartbeatTTL {
		return responses.Heartbeat{}, false
	}
	return entry.result, true
}

// probeStatus maps a probe onto the three states a heartbeat can report.
//
// The distinction that matters is between "ran and found it down" and "could
// not run": a cancelled request or a malformed target says nothing about the
// service, and reporting it as down would raise a false alarm.
func probeStatus(probe clients.ProbeResult) string {
	switch {
	case probe.Serving:
		return constants.StatusHealthy
	case probe.Inconclusive:
		// Ran, but learned nothing about the service — an unresolvable target is
		// a services.yaml problem, and the placeholders shipped there would
		// otherwise show the fleet as down on day one.
		return constants.StatusUnknown
	case probe.Err != nil && probe.Latency == 0:
		// Never left the ground — config or context, not the service.
		return constants.StatusUnknown
	default:
		return constants.StatusFailed
	}
}

// heartbeatReason explains a status the heartbeat decided, naming the probe so
// it is clear the verdict did not come from Airflow.
func heartbeatReason(hb *responses.Heartbeat) *string {
	if hb == nil || hb.Status != constants.StatusFailed {
		return nil
	}
	msg := "heartbeat failing"
	if hb.Detail != nil {
		msg = "heartbeat failing — " + *hb.Detail
	}
	return &msg
}

// ApplyHeartbeat folds a heartbeat into the status derived from a service's
// DAGs.
//
// The heartbeat is authoritative for "down": if the thing is not answering, no
// amount of green DAG history makes it healthy. It is not authoritative for
// "healthy" — a service answering its health endpoint while its DAGs fail is
// degraded, and must keep saying so.
func ApplyHeartbeat(dagStatus string, hb *responses.Heartbeat) string {
	if hb == nil {
		return dagStatus
	}
	switch hb.Status {
	case constants.StatusFailed:
		return constants.StatusFailed
	case constants.StatusHealthy:
		// Services with no DAGs are exactly the ones heartbeats exist for; the
		// probe is the only evidence there is, so let it decide.
		if dagStatus == constants.StatusUnknown {
			return constants.StatusHealthy
		}
		return dagStatus
	default:
		return dagStatus
	}
}
