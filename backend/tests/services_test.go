package tests

import (
	"context"
	"slices"
	"testing"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/models"
	"github.com/openclimatefix/hexatron/backend/internal/services"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
	"github.com/openclimatefix/hexatron/backend/internal/structures/responses"
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
	if consumerSvc.Name != "Consumers" {
		t.Errorf("expected service name Consumers, got %s", consumerSvc.Name)
	}
	if len(consumerSvc.DAGPatterns) == 0 {
		t.Errorf("expected 'consumer' service to define DAG patterns")
	}

	_, notFound := registry.ByID("non-existent-id")
	if notFound {
		t.Errorf("expected non-existent service ID to return false")
	}
}

// TestServiceRegistryRejectsBadConfig covers the services.yaml errors that would otherwise pass silently.
func TestServiceRegistryRejectsBadConfig(t *testing.T) {
	cases := []struct {
		name string
		yaml string
	}{
		{"malformed dag pattern", `
services:
  - id: forecast
    name: Forecast
    dag_patterns:
      - "uk-[forecast"
`},
		{"dangling depends_on", `
services:
  - id: forecast
    name: Forecast
    depends_on:
      - no-such-service
`},
		{"duplicate service id", `
services:
  - id: forecast
    name: Forecast
  - id: forecast
    name: Forecast Again
`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := services.NewServiceRegistry(writeServicesYAML(t, tc.yaml)); err == nil {
				t.Error("expected an error loading the registry, got nil")
			}
		})
	}
}

// TestMatchDAGsIsAnchored checks a forecast service does not claim nl-consume-ned-nl-forecast.
func TestMatchDAGsIsAnchored(t *testing.T) {
	airflowDAGs := []string{
		"nl-consume-ned-nl-forecast",
		"nl-forecast",
		"uk-analysis-clouds",
		"uk-consume-pv",
		"uk-forecast-site",
	}

	forecast := models.Service{DAGPatterns: []string{"uk-forecast-*", "uk-analysis-*", "nl-forecast"}}
	want := []string{"nl-forecast", "uk-analysis-clouds", "uk-forecast-site"}

	got := forecast.MatchDAGs(airflowDAGs)
	if !slices.Equal(got, want) {
		t.Errorf("MatchDAGs() = %v, want %v", got, want)
	}

	// A service with no patterns has no DAGs and aggregates to unknown.
	empty := models.Service{ID: "wind-forecast"}
	if got := empty.MatchDAGs(airflowDAGs); len(got) != 0 {
		t.Errorf("a service with no patterns claimed %v", got)
	}
	if got := services.AggregateStatus(nil); got != constants.StatusUnknown {
		t.Errorf("a service with no DAGs = %q, want %q", got, constants.StatusUnknown)
	}
}

// ocfAirflowDAGs is a snapshot of the DAGs the OCF Airflow deployment serves.
var ocfAirflowDAGs = []string{
	"nl-api-check",
	"nl-consume-ned-nl",
	"nl-consume-ned-nl-forecast",
	"nl-consume-nwp",
	"nl-forecast",
	"uk-analysis-clouds",
	"uk-api-quartz-national-gsp-check",
	"uk-api-site-check",
	"uk-consume-neso",
	"uk-consume-nwp",
	"uk-consume-pv",
	"uk-consume-pvlive-dayafter",
	"uk-consume-pvlive-intraday",
	"uk-consume-sat",
	"uk-consume-sat-v1",
	"uk-forecast-clouds",
	"uk-forecast-gsp",
	"uk-forecast-site",
	"uk-manage-clean-up-logs",
	"uk-manage-elb",
	"uk-manage-sitedb-cleanup",
}

func TestShippedConfigRoutesOCFDAGs(t *testing.T) {
	registry, err := services.NewServiceRegistry(constants.ServicesConfigPath)
	if err != nil {
		t.Fatalf("unexpected error loading service registry: %v", err)
	}

	want := map[string][]string{
		"solar-forecast": {
			"nl-forecast",
			"uk-analysis-clouds",
			"uk-forecast-clouds",
			"uk-forecast-gsp",
			"uk-forecast-site",
		},
		// No wind DAGs in Airflow yet, and no UI monitoring.
		"wind-forecast": {},
		"ui":            {},
		"consumer": {
			"nl-consume-ned-nl",
			"nl-consume-ned-nl-forecast",
			"nl-consume-nwp",
			"uk-consume-neso",
			"uk-consume-nwp",
			"uk-consume-pv",
			"uk-consume-pvlive-dayafter",
			"uk-consume-pvlive-intraday",
			"uk-consume-sat",
			"uk-consume-sat-v1",
		},
		"data-platform": {
			"uk-manage-clean-up-logs",
			"uk-manage-elb",
			"uk-manage-sitedb-cleanup",
		},
		"api": {
			"nl-api-check",
			"uk-api-quartz-national-gsp-check",
			"uk-api-site-check",
		},
	}

	all := registry.All()
	if len(all) != len(want) {
		t.Fatalf("expected %d services in the registry, got %d", len(want), len(all))
	}

	claimed := make(map[string]string, len(ocfAirflowDAGs))
	for _, svc := range all {
		expected, known := want[svc.ID]
		if !known {
			t.Errorf("unexpected service %q in registry", svc.ID)
			continue
		}

		got := svc.MatchDAGs(ocfAirflowDAGs)
		if !slices.Equal(got, expected) {
			t.Errorf("%s claimed %v, want %v", svc.ID, got, expected)
		}

		for _, dagID := range got {
			if other, taken := claimed[dagID]; taken {
				t.Errorf("%s is claimed by both %s and %s", dagID, other, svc.ID)
			}
			claimed[dagID] = svc.ID
		}
	}

	// Every DAG the deployment runs belongs to exactly one service.
	for _, dagID := range ocfAirflowDAGs {
		if _, ok := claimed[dagID]; !ok {
			t.Errorf("%s is claimed by no service", dagID)
		}
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

// testServicesYAML defines two services whose patterns claim four DAGs between them.
const testServicesYAML = `
services:
  - id: forecast
    name: Forecast
    category: Forecast
    dag_patterns:
      - forecast-*
  - id: consumer
    name: Consumer
    category: Consumer
    depends_on:
      - forecast
    dag_patterns:
      - consume-*
`

func newTestService(t *testing.T, cookie string) *services.AirflowService {
	t.Helper()

	airflow := startFakeAirflow(t, map[string]fakeAirflowDAG{
		"forecast-ok":       {State: constants.AirflowStateSuccess},
		"forecast-running":  {State: constants.AirflowStateRunning},
		"consume-broken":    {State: constants.AirflowStateFailed},
		"consume-never-run": {State: ""},
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

	// A running DAG must not be reported as failed.
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

	byID := make(map[string]responses.DAGDetail, len(detail.DAGs))
	for _, d := range detail.DAGs {
		byID[d.DAGID] = d
	}

	broken := byID["consume-broken"]
	if broken.Status != constants.StatusFailed {
		t.Errorf("consume-broken: expected %q, got %q", constants.StatusFailed, broken.Status)
	}
	if broken.LastRun == nil {
		t.Error("consume-broken: expected a last run")
	}
	if broken.AirflowURL == "" {
		t.Error("consume-broken: expected an Airflow deep link")
	}
	if broken.Schedule == "" {
		t.Error("consume-broken: expected a schedule")
	}

	// A DAG that has never run has no last run, and is unknown rather than failed.
	neverRun := byID["consume-never-run"]
	if neverRun.Status != constants.StatusUnknown {
		t.Errorf("consume-never-run: expected %q, got %q", constants.StatusUnknown, neverRun.Status)
	}
	if neverRun.LastRun != nil {
		t.Errorf("consume-never-run: expected no last run, got %+v", neverRun.LastRun)
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

// TestExpiredCookieSurfacesError checks an authentication failure is reported as an error.
func TestExpiredCookieSurfacesError(t *testing.T) {
	svc := newTestService(t, "")

	if _, err := svc.ListServices(context.Background(), "", ""); err == nil {
		t.Fatal("expected an error when the session cookie is missing")
	}
}

func TestConfigDriftDetectsUnknownDAG(t *testing.T) {
	airflow := startFakeAirflow(t, map[string]fakeAirflowDAG{
		"forecast-ok":  {State: constants.AirflowStateSuccess},
		"dag-orphaned": {State: constants.AirflowStateSuccess},
	})

	// wind has no patterns and is not drift; the typo'd pattern is.
	svc := services.NewAirflowService(&configstructs.Config{
		ServicesConfigPath: writeServicesYAML(t, `
services:
  - id: forecast
    name: Forecast
    category: Forecast
    dag_patterns:
      - forecast-*
      - typo-*
  - id: wind
    name: Wind
    category: Forecast
    dag_patterns: []
`),
		AirflowBaseURL: airflow.URL,
		AirflowCookie:  "test-cookie",
	})

	unmatched, unclaimed, err := svc.ConfigDrift(context.Background())
	if err != nil {
		t.Fatalf("ConfigDrift returned error: %v", err)
	}

	if len(unmatched) != 1 || unmatched[0] != "forecast: typo-*" {
		t.Errorf("expected the typo-* pattern reported unmatched, got %v", unmatched)
	}
	if len(unclaimed) != 1 || unclaimed[0] != "dag-orphaned" {
		t.Errorf("expected dag-orphaned reported unclaimed, got %v", unclaimed)
	}
}
