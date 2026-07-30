// Package mock defines the in-memory mock data structures used in Phase 1.
// In Phase 2, replace MockServiceStore with live Airflow API calls.
package mock

import "github.com/openclimatefix/hexatron/backend/structures/responses"

// MockService represents a single business service entry in the mock dataset.
// Its shape mirrors the structure defined in data/services.yaml.
type MockService struct {
	ID       string
	Name     string
	Category string
	DAGs     []responses.DAGStatus
}

// MockServiceStore is the in-memory dataset backing all service endpoints.
var MockServiceStore = []MockService{
	{
		ID:       "site-forecast",
		Name:     "Site Forecast",
		Category: "Forecast",
		DAGs: []responses.DAGStatus{
			{DAGID: "site_forecast", Status: "healthy"},
		},
	},
	{
		ID:       "consumer",
		Name:     "Consumer",
		Category: "Consumer",
		DAGs: []responses.DAGStatus{
			{DAGID: "ecmwf_consumer", Status: "healthy"},
			{DAGID: "metoffice_consumer", Status: "failed"},
			{DAGID: "pvlive_consumer", Status: "healthy"},
		},
	},
	{
		ID:       "data-platform",
		Name:     "Data Platform",
		Category: "Platform",
		DAGs: []responses.DAGStatus{
			{DAGID: "save_to_dp", Status: "healthy"},
		},
	},
}
