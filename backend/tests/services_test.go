package tests

import (
	"context"
	"testing"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/models"
	"github.com/openclimatefix/hexatron/backend/internal/services"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
)

func TestServiceRegistry(t *testing.T) {
	registry, err := services.NewServiceRegistry(constants.ServicesConfigPath)
	if err != nil {
		t.Fatalf("unexpected error loading service registry: %v", err)
	}

	allServices := registry.All()
	if len(allServices) == 0 {
		t.Fatal("expected at least 1 service in registry, got 0")
	}

	consumerSvc, found := registry.ByID("consumer")
	if !found {
		t.Fatalf("expected to find 'consumer' service by ID")
	}
	if consumerSvc.Name != "Consumer" {
		t.Errorf("expected service name Consumer, got %s", consumerSvc.Name)
	}
	if len(consumerSvc.DAGIDs) == 0 {
		t.Errorf("expected 'consumer' service to list DAGs")
	}

	_, notFound := registry.ByID("non-existent-id")
	if notFound {
		t.Errorf("expected non-existent service ID to return false")
	}
}

func TestMapAirflowStateToStatus(t *testing.T) {
	cases := []struct {
		state string
		want  string
	}{
		{constants.AirflowStateSuccess, constants.StatusHealthy},
		{constants.AirflowStateFailed, constants.StatusFailed},
		{constants.AirflowStateUpstreamFailed, constants.StatusFailed},
		{constants.AirflowStateRunning, constants.StatusRunning},
		{constants.AirflowStateQueued, constants.StatusQueued},
		{"", constants.StatusUnknown},
		{"something_new", constants.StatusUnknown},
	}

	for _, tc := range cases {
		if got := services.MapAirflowStateToStatus(tc.state); got != tc.want {
			t.Errorf("MapAirflowStateToStatus(%q) = %q, want %q", tc.state, got, tc.want)
		}
	}
}

func TestAggregateStatus(t *testing.T) {
	dags := func(statuses ...string) []models.DAGStatus {
		out := make([]models.DAGStatus, len(statuses))
		for i, s := range statuses {
			out[i] = models.DAGStatus{DAGID: "dag", Status: s}
		}
		return out
	}

	cases := []struct {
		name string
		in   []models.DAGStatus
		want string
	}{
		{"no dags is unknown", nil, constants.StatusUnknown},
		{"all healthy", dags(constants.StatusHealthy, constants.StatusHealthy), constants.StatusHealthy},
		{"any failed wins", dags(constants.StatusHealthy, constants.StatusFailed), constants.StatusFailed},
		{"failure outranks running", dags(constants.StatusRunning, constants.StatusFailed), constants.StatusFailed},
		{"failure outranks unknown", dags(constants.StatusUnknown, constants.StatusFailed), constants.StatusFailed},
		{"running over queued", dags(constants.StatusQueued, constants.StatusRunning), constants.StatusRunning},
		{"running over healthy", dags(constants.StatusHealthy, constants.StatusRunning), constants.StatusRunning},
		{"queued over healthy", dags(constants.StatusHealthy, constants.StatusQueued), constants.StatusQueued},
		{"unknown over healthy", dags(constants.StatusHealthy, constants.StatusUnknown), constants.StatusUnknown},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := services.AggregateStatus(tc.in); got != tc.want {
				t.Errorf("AggregateStatus() = %q, want %q", got, tc.want)
			}
		})
	}
}

// testServicesYAML defines two services over four DAGs, enough to cover a
// healthy service and a service broken by a single DAG.
const testServicesYAML = `
services:
  - id: forecast
    name: Forecast
    category: Forecast
    dags:
      - dag-ok
      - dag-running
  - id: consumer
    name: Consumer
    category: Consumer
    dags:
      - dag-broken
      - dag-never-run
`

func newTestService(t *testing.T, cookie string) *services.AirflowService {
	t.Helper()

	airflow := startFakeAirflow(t, map[string]fakeAirflowDAG{
		"dag-ok":        {State: constants.AirflowStateSuccess},
		"dag-running":   {State: constants.AirflowStateRunning},
		"dag-broken":    {State: constants.AirflowStateFailed},
		"dag-never-run": {State: ""},
	})

	return services.NewAirflowService(&configstructs.Config{
		ServicesConfigPath: writeServicesYAML(t, testServicesYAML),
		AirflowBaseURL:     airflow.URL,
		AirflowCookie:      cookie,
	})
}

func TestListServicesAggregatesRealRunStates(t *testing.T) {
	svc := newTestService(t, "test-cookie")

	summaries, err := svc.ListServices(context.Background(), "", "")
	if err != nil {
		t.Fatalf("ListServices returned error: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 services, got %d", len(summaries))
	}

	got := make(map[string]string, len(summaries))
	for _, s := range summaries {
		got[s.ID] = s.Status
	}

	// A running DAG must not be reported as failed: that was the behaviour when
	// running collapsed into unknown.
	if got["forecast"] != constants.StatusRunning {
		t.Errorf("forecast: expected %q, got %q", constants.StatusRunning, got["forecast"])
	}
	if got["consumer"] != constants.StatusFailed {
		t.Errorf("consumer: expected %q, got %q", constants.StatusFailed, got["consumer"])
	}
}

func TestListServicesFilters(t *testing.T) {
	svc := newTestService(t, "test-cookie")
	ctx := context.Background()

	bySearch, err := svc.ListServices(ctx, "consum", "")
	if err != nil {
		t.Fatalf("ListServices(search) returned error: %v", err)
	}
	if len(bySearch) != 1 || bySearch[0].ID != "consumer" {
		t.Errorf("search=consum: expected only the consumer service, got %+v", bySearch)
	}

	byCategory, err := svc.ListServices(ctx, "", "Forecast")
	if err != nil {
		t.Fatalf("ListServices(category) returned error: %v", err)
	}
	if len(byCategory) != 1 || byCategory[0].ID != "forecast" {
		t.Errorf("category=Forecast: expected only the forecast service, got %+v", byCategory)
	}

	none, err := svc.ListServices(ctx, "no-such-service", "")
	if err != nil {
		t.Fatalf("ListServices(no match) returned error: %v", err)
	}
	if len(none) != 0 {
		t.Errorf("expected no matches, got %d", len(none))
	}
}

func TestGetServiceByIDReturnsDAGDetail(t *testing.T) {
	svc := newTestService(t, "test-cookie")

	detail, found, err := svc.GetServiceByID(context.Background(), "consumer")
	if err != nil {
		t.Fatalf("GetServiceByID returned error: %v", err)
	}
	if !found {
		t.Fatal("expected service 'consumer' to be found")
	}
	if detail.Status != constants.StatusFailed {
		t.Errorf("expected consumer status %q, got %q", constants.StatusFailed, detail.Status)
	}
	if len(detail.DAGs) != 2 {
		t.Fatalf("expected 2 DAGs, got %d", len(detail.DAGs))
	}

	byID := make(map[string]models.DAGStatus, len(detail.DAGs))
	for _, d := range detail.DAGs {
		byID[d.DAGID] = d
	}

	broken := byID["dag-broken"]
	if broken.Status != constants.StatusFailed {
		t.Errorf("dag-broken: expected %q, got %q", constants.StatusFailed, broken.Status)
	}
	if broken.LastRun == nil {
		t.Error("dag-broken: expected a last run")
	}
	if broken.AirflowURL == "" {
		t.Error("dag-broken: expected an Airflow deep link")
	}
	if broken.Schedule == "" {
		t.Error("dag-broken: expected a schedule")
	}

	// A DAG that has never run has no last run, and is unknown rather than failed.
	neverRun := byID["dag-never-run"]
	if neverRun.Status != constants.StatusUnknown {
		t.Errorf("dag-never-run: expected %q, got %q", constants.StatusUnknown, neverRun.Status)
	}
	if neverRun.LastRun != nil {
		t.Errorf("dag-never-run: expected no last run, got %+v", neverRun.LastRun)
	}
}

func TestGetServiceByIDNotFound(t *testing.T) {
	svc := newTestService(t, "test-cookie")

	_, found, err := svc.GetServiceByID(context.Background(), "invalid-service")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found {
		t.Error("expected invalid service to return false")
	}
}

// TestExpiredCookieSurfacesError checks that an authentication failure is
// reported as an error rather than being flattened into "everything is broken".
func TestExpiredCookieSurfacesError(t *testing.T) {
	svc := newTestService(t, "")

	if _, err := svc.ListServices(context.Background(), "", ""); err == nil {
		t.Fatal("expected an error when the session cookie is missing")
	}
}

func TestConfigDriftDetectsUnknownDAG(t *testing.T) {
	airflow := startFakeAirflow(t, map[string]fakeAirflowDAG{
		"dag-ok":       {State: constants.AirflowStateSuccess},
		"dag-orphaned": {State: constants.AirflowStateSuccess},
	})

	svc := services.NewAirflowService(&configstructs.Config{
		ServicesConfigPath: writeServicesYAML(t, `
services:
  - id: forecast
    name: Forecast
    category: Forecast
    dags:
      - dag-ok
      - dag-typo
`),
		AirflowBaseURL: airflow.URL,
		AirflowCookie:  "test-cookie",
	})

	missing, unclaimed, err := svc.ConfigDrift(context.Background())
	if err != nil {
		t.Fatalf("ConfigDrift returned error: %v", err)
	}

	if len(missing) != 1 || missing[0] != "dag-typo" {
		t.Errorf("expected dag-typo reported missing, got %v", missing)
	}
	if len(unclaimed) != 1 || unclaimed[0] != "dag-orphaned" {
		t.Errorf("expected dag-orphaned reported unclaimed, got %v", unclaimed)
	}
}
