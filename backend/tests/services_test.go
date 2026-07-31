package tests

import (
	"testing"

	"github.com/openclimatefix/hexatron/backend/constants"
	"github.com/openclimatefix/hexatron/backend/services"
	clientstructs "github.com/openclimatefix/hexatron/backend/structures/clients"
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
	clientCfg := clientstructs.AirflowClientConfig{
		BaseURL: "http://127.0.0.1:38000",
		Cookie:  "",
	}
	airflowService := services.NewAirflowService(constants.ServicesConfigPath, clientCfg)

	// Test ListServices
	listResp := airflowService.ListServices("", "")
	if len(listResp) == 0 {
		t.Fatal("expected listResp to return services, got empty")
	}

	// Test GetServiceByID
	detailResp, found := airflowService.GetServiceByID("forecasts")
	if !found {
		t.Fatalf("expected service 'forecasts' to be found")
	}
	if detailResp.ID != "forecasts" {
		t.Errorf("expected ID 'forecasts', got %s", detailResp.ID)
	}

	_, notFound := airflowService.GetServiceByID("invalid-service")
	if notFound {
		t.Errorf("expected invalid service to return false")
	}
}
