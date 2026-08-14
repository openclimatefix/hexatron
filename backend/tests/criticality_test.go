package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/models"
	"github.com/openclimatefix/hexatron/backend/internal/routes"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
	"github.com/openclimatefix/hexatron/backend/internal/structures/responses"
)

func TestIsCritical(t *testing.T) {
	svc := models.Service{
		DAGPatterns:            []string{"uk-forecast-*", "uk-analysis-*"},
		NonCriticalDAGPatterns: []string{"uk-forecast-clouds", "uk-analysis-clouds"},
	}

	cases := map[string]bool{
		"uk-forecast-clouds":  false,
		"uk-analysis-clouds":  false,
		"uk-forecast-solar":   true,
		"uk-analysis-solar":   true,
		"something-unclaimed": true,
	}

	for dagID, want := range cases {
		if got := svc.IsCritical(dagID); got != want {
			t.Errorf("IsCritical(%q) = %v, want %v", dagID, got, want)
		}
	}
}

// The bug this exists to prevent: one failing cloudcasting DAG reddening the
// whole Solar Forecast service.
func TestNonCriticalFailureDegradesRatherThanDowns(t *testing.T) {
	byID := servicesFrom(t, map[string]fakeAirflowDAG{
		"uk-forecast-solar":  {State: constants.AirflowStateSuccess},
		"uk-forecast-clouds": {State: constants.AirflowStateFailed},
	})

	solar := byID["solar-forecast"]
	if solar.Status != constants.StatusDegraded {
		t.Errorf("expected %q when only cloudcasting fails, got %q",
			constants.StatusDegraded, solar.Status)
	}
	// Degraded but not silent — the operator must be told which DAG.
	if solar.StatusReason == nil {
		t.Fatal("expected a status_reason naming the failing DAG")
	}
	if want := "uk-forecast-clouds failing (non-critical)"; *solar.StatusReason != want {
		t.Errorf("expected reason %q, got %q", want, *solar.StatusReason)
	}
}

// The severity narrowing must not swallow real failures.
func TestCriticalFailureStillDowns(t *testing.T) {
	byID := servicesFrom(t, map[string]fakeAirflowDAG{
		"uk-forecast-solar":  {State: constants.AirflowStateFailed},
		"uk-forecast-clouds": {State: constants.AirflowStateSuccess},
	})

	solar := byID["solar-forecast"]
	if solar.Status != constants.StatusFailed {
		t.Errorf("expected %q when the solar DAG fails, got %q",
			constants.StatusFailed, solar.Status)
	}
	if solar.StatusReason == nil || *solar.StatusReason != "uk-forecast-solar failing" {
		t.Errorf("expected the failing DAG to be named, got %v", solar.StatusReason)
	}
}

// A critical failure alongside a non-critical one is still down: the worst
// verdict wins, and the reason names the one that caused it.
func TestCriticalWinsOverNonCritical(t *testing.T) {
	byID := servicesFrom(t, map[string]fakeAirflowDAG{
		"uk-forecast-solar":  {State: constants.AirflowStateFailed},
		"uk-forecast-clouds": {State: constants.AirflowStateFailed},
	})

	solar := byID["solar-forecast"]
	if solar.Status != constants.StatusFailed {
		t.Errorf("expected %q, got %q", constants.StatusFailed, solar.Status)
	}
	if solar.StatusReason == nil || *solar.StatusReason != "uk-forecast-solar failing" {
		t.Errorf("expected only the critical DAG named, got %v", solar.StatusReason)
	}
}

func TestHealthyServiceHasNoReason(t *testing.T) {
	byID := servicesFrom(t, map[string]fakeAirflowDAG{
		"uk-forecast-solar":  {State: constants.AirflowStateSuccess},
		"uk-forecast-clouds": {State: constants.AirflowStateSuccess},
	})

	solar := byID["solar-forecast"]
	if solar.Status != constants.StatusHealthy {
		t.Errorf("expected %q, got %q", constants.StatusHealthy, solar.Status)
	}
	if solar.StatusReason != nil {
		t.Errorf("expected no reason on a healthy service, got %q", *solar.StatusReason)
	}
}

// servicesFrom runs /services against a fake Airflow holding the given DAGs,
// using a config that mirrors the real solar-forecast criticality setup.
func servicesFrom(t *testing.T, dags map[string]fakeAirflowDAG) map[string]responses.ServiceResponse {
	t.Helper()

	airflow := startFakeAirflow(t, dags)

	yaml := "services:\n" +
		"  - id: solar-forecast\n" +
		"    name: Solar Forecast\n" +
		"    category: Forecast\n" +
		"    depends_on: []\n" +
		"    dag_patterns:\n" +
		"      - uk-forecast-*\n" +
		"    non_critical_dag_patterns:\n" +
		"      - uk-forecast-clouds\n"

	router := routes.NewRouter(&configstructs.Config{
		Addr:               ":8080",
		ServicesConfigPath: writeServicesYAML(t, yaml),
		AirflowBaseURL:     airflow.URL,
		AirflowCookie:      "test-cookie",
	})

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var got responses.ServiceListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("could not decode response: %v", err)
	}

	byID := make(map[string]responses.ServiceResponse, len(got))
	for _, svc := range got {
		byID[svc.ID] = svc
	}
	return byID
}
