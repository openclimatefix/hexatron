package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/openclimatefix/hexatron/backend/internal/config"
	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/routes"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
	"github.com/openclimatefix/hexatron/backend/internal/structures/responses"
)

func TestMain(m *testing.M) {
	// Ensure working directory is backend root so relative data paths resolve correctly
	if _, err := os.Stat("data/services.yaml"); os.IsNotExist(err) {
		if _, err := os.Stat("../data/services.yaml"); err == nil {
			_ = os.Chdir("..")
		}
	}
	os.Exit(m.Run())
}

// newTestRouter builds the real router against a fake Airflow, so the API tests
// exercise the full stack without needing a live Airflow.
func newTestRouter(t *testing.T) http.Handler {
	t.Helper()

	airflow := startFakeAirflow(t, map[string]fakeAirflowDAG{
		"forecast-ok":       {State: constants.AirflowStateSuccess},
		"forecast-running":  {State: constants.AirflowStateRunning},
		"consume-broken":    {State: constants.AirflowStateFailed},
		"consume-never-run": {State: ""},
	})

	return routes.NewRouter(&configstructs.Config{
		Addr:               ":8080",
		ServicesConfigPath: writeServicesYAML(t, testServicesYAML),
		AirflowBaseURL:     airflow.URL,
		AirflowCookie:      "test-cookie",
	})
}

func TestHealthEndpoint(t *testing.T) {
	cfg := config.Load()
	router := routes.NewRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res responses.HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode health response: %v", err)
	}

	if res.Status != "healthy" {
		t.Errorf("expected status 'healthy', got %s", res.Status)
	}
}

func TestListServicesEndpoint(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	var res responses.ServiceListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode services response: %v", err)
	}

	if len(res) == 0 {
		t.Fatal("expected non-empty list of services")
	}

	for _, svc := range res {
		if svc.ID == "" || svc.Name == "" || svc.Status == "" {
			t.Errorf("service is missing required fields: %+v", svc)
		}
	}
}

func TestListServicesEndpointFilters(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/services?category=Consumer", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res responses.ServiceListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode services response: %v", err)
	}

	if len(res) != 1 || res[0].ID != "consumer" {
		t.Errorf("expected only the consumer service, got %+v", res)
	}
}

func TestGetServiceByIDEndpoint(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/services/consumer", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /services/consumer, got %d: %s", rec.Code, rec.Body.String())
	}

	var res responses.ServiceDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode service detail response: %v", err)
	}
	if res.ID != "consumer" {
		t.Errorf("expected service ID 'consumer', got %s", res.ID)
	}
	if res.Status != constants.StatusFailed {
		t.Errorf("expected status %q, got %q", constants.StatusFailed, res.Status)
	}
	if len(res.DAGs) == 0 {
		t.Fatal("expected the detail response to list DAGs")
	}

	// The detail response carries the context needed to act on a failure.
	for _, dag := range res.DAGs {
		if dag.DAGID == "" || dag.Status == "" {
			t.Errorf("dag entry missing dag_id or status: %+v", dag)
		}
		if dag.AirflowURL == "" {
			t.Errorf("dag %s: expected an airflow_url deep link", dag.DAGID)
		}
	}
}

func TestGetServiceByIDNotFoundEndpoint(t *testing.T) {
	router := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/services/non-existent-id", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for invalid service ID, got %d", rec.Code)
	}

	var res responses.ErrorResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode error response: %v", err)
	}
	if res.Error == "" {
		t.Error("expected an error message in the 404 response")
	}
}

// TestAirflowUnreachableReturnsBadGateway checks that a broken Airflow surfaces
// as an upstream error, not as every service reporting failed.
func TestAirflowUnreachableReturnsBadGateway(t *testing.T) {
	router := routes.NewRouter(&configstructs.Config{
		Addr:               ":8080",
		ServicesConfigPath: writeServicesYAML(t, testServicesYAML),
		// Port 1 is reserved and never listening.
		AirflowBaseURL: "http://127.0.0.1:1",
		AirflowCookie:  "test-cookie",
	})

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadGateway {
		t.Fatalf("expected status 502 when Airflow is unreachable, got %d", rec.Code)
	}
}
