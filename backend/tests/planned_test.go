package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/routes"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
	"github.com/openclimatefix/hexatron/backend/internal/structures/responses"
)

// A planned service with no DAGs is genuinely "not built yet", not "unknown".
func TestPlannedServiceWithNoDAGsReportsPlanned(t *testing.T) {
	byID := plannedServicesFrom(t, map[string]fakeAirflowDAG{
		"uk-forecast-solar": {State: constants.AirflowStateSuccess},
	})

	wind := byID["wind-forecast"]
	if wind.Status != constants.StatusPlanned {
		t.Errorf("expected %q for a planned service with no DAGs, got %q",
			constants.StatusPlanned, wind.Status)
	}
	if wind.StatusReason != nil {
		t.Errorf("expected no reason for a planned service, got %q", *wind.StatusReason)
	}
}

// planned: true must not override a service that actually has DAGs — it only
// applies to the no-DAGs case.
func TestPlannedDoesNotOverrideServiceWithDAGs(t *testing.T) {
	byID := plannedServicesFrom(t, map[string]fakeAirflowDAG{
		"uk-forecast-solar": {State: constants.AirflowStateSuccess},
	})

	solar := byID["solar-forecast"]
	if solar.Status == constants.StatusPlanned {
		t.Errorf("planned:true must not override a service with DAGs, got %q", solar.Status)
	}
	if solar.Status != constants.StatusHealthy {
		t.Errorf("expected %q, got %q", constants.StatusHealthy, solar.Status)
	}
}

// plannedServicesFrom runs /services against a fake Airflow, with one
// planned service that has no DAGs and one planned service that does.
func plannedServicesFrom(t *testing.T, dags map[string]fakeAirflowDAG) map[string]responses.ServiceResponse {
	t.Helper()

	airflow := startFakeAirflow(t, dags)

	yaml := "services:\n" +
		"  - id: wind-forecast\n" +
		"    name: Wind Forecast\n" +
		"    category: Forecast\n" +
		"    depends_on: []\n" +
		"    dag_patterns: []\n" +
		"    planned: true\n" +
		"  - id: solar-forecast\n" +
		"    name: Solar Forecast\n" +
		"    category: Forecast\n" +
		"    depends_on: []\n" +
		"    dag_patterns:\n" +
		"      - uk-forecast-*\n" +
		"    planned: true\n"

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
