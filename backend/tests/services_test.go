package tests

import (
	"testing"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
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

	forecastsSvc, found := registry.ByID("forecasts")
	if !found {
		t.Errorf("expected to find 'forecasts' service by ID")
	}
	if forecastsSvc.Name != "Forecasts" {
		t.Errorf("expected service name Forecasts, got %s", forecastsSvc.Name)
	}

	_, notFound := registry.ByID("non-existent-id")
	if notFound {
		t.Errorf("expected non-existent service ID to return false")
	}
}

func TestAirflowServiceListAndGet(t *testing.T) {
	cfg := &configstructs.Config{
		ServicesConfigPath: constants.ServicesConfigPath,
		AirflowBaseURL:     constants.AirflowDefaultURL,
		AirflowCookie:      "",
	}
	airflowService := services.NewAirflowService(cfg)

	// Test ListServices
	listResp := airflowService.ListServices("", "")
	if len(listResp) == 0 {
		t.Fatal("expected ListServices to return services, got empty")
	}

	// Test GetServiceByID - found
	detail, found := airflowService.GetServiceByID("forecasts")
	if !found {
		t.Fatalf("expected service 'forecasts' to be found")
	}
	if detail.ID != "forecasts" {
		t.Errorf("expected ID 'forecasts', got %s", detail.ID)
	}
	if len(detail.DAGs) == 0 {
		t.Errorf("expected detail.DAGs to be non-empty for 'forecasts'")
	}

	// Test GetServiceByID - not found
	_, notFound := airflowService.GetServiceByID("invalid-service")
	if notFound {
		t.Errorf("expected invalid service to return false")
	}
}
