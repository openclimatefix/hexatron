package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/openclimatefix/hexatron/backend/internal/config"
	"github.com/openclimatefix/hexatron/backend/internal/routes"
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
	cfg := config.Load()
	router := routes.NewRouter(cfg)

	req := httptest.NewRequest(http.MethodGet, "/services", nil)
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res responses.ServiceListResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode services response: %v", err)
	}

	if len(res) == 0 {
		t.Errorf("expected non-empty list of services")
	}
}

func TestGetServiceByIDEndpoint(t *testing.T) {
	cfg := config.Load()
	router := routes.NewRouter(cfg)

	// Valid service
	req := httptest.NewRequest(http.MethodGet, "/services/forecasts", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200 for /services/forecasts, got %d", rec.Code)
	}

	var res responses.ServiceDetailResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("failed to decode service detail response: %v", err)
	}
	if res.ID != "forecasts" {
		t.Errorf("expected service ID 'forecasts', got %s", res.ID)
	}

	// Invalid service
	reqNotFound := httptest.NewRequest(http.MethodGet, "/services/non-existent-id", nil)
	recNotFound := httptest.NewRecorder()
	router.ServeHTTP(recNotFound, reqNotFound)

	if recNotFound.Code != http.StatusNotFound {
		t.Errorf("expected status 404 for invalid service ID, got %d", recNotFound.Code)
	}
}
