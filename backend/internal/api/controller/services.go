// Package controller handles incoming HTTP requests and returns mock responses.
// Each function maps 1:1 to a REST endpoint.
//
// Mock data is defined inline here. In a future phase these functions will
// delegate to the services layer which calls the Airflow client.
package controller

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/openclimatefix/hexatron/backend/internal/structures"
)

// mockServices is the in-memory mock dataset that backs both endpoints.
// Structure mirrors services.yaml so it is easy to keep in sync.
var mockServices = []struct {
	ID       string
	Name     string
	Category string
	DAGs     []structures.DAGStatus
}{
	{
		ID:       "site-forecast",
		Name:     "Site Forecast",
		Category: "Forecast",
		DAGs: []structures.DAGStatus{
			{DAGID: "site_forecast", Status: "healthy"},
		},
	},
	{
		ID:       "consumer",
		Name:     "Consumer",
		Category: "Consumer",
		DAGs: []structures.DAGStatus{
			{DAGID: "ecmwf_consumer", Status: "healthy"},
			{DAGID: "metoffice_consumer", Status: "failed"},
			{DAGID: "pvlive_consumer", Status: "healthy"},
		},
	},
	{
		ID:       "data-platform",
		Name:     "Data Platform",
		Category: "Platform",
		DAGs: []structures.DAGStatus{
			{DAGID: "save_to_dp", Status: "healthy"},
		},
	},
}

// aggregateStatus derives the overall service status from its DAGs.
// Returns "failed" if any DAG is failed, otherwise "healthy".
func aggregateStatus(dags []structures.DAGStatus) string {
	for _, d := range dags {
		if d.Status == "failed" {
			return "failed"
		}
	}
	return "healthy"
}

// ListServices handles GET /services.
//
// Request payload:  structures.GetServicesRequestPayload  (query params)
// Response payload: structures.GetServicesResponsePayload (JSON array)
//
// Optional query params:
//   - ?search=<string>   case-insensitive name filter
//   - ?category=<string> exact category match
//
// Mock response:
//
//	[
//	  { "id": "site-forecast", "name": "Site Forecast", "status": "healthy" },
//	  { "id": "consumer",      "name": "Consumer",      "status": "failed"  },
//	  { "id": "data-platform", "name": "Data Platform", "status": "healthy" }
//	]
func ListServices(w http.ResponseWriter, r *http.Request) {
	req := structures.GetServicesRequestPayload{
		Search:   r.URL.Query().Get("search"),
		Category: r.URL.Query().Get("category"),
	}

	response := make(structures.GetServicesResponsePayload, 0)

	for _, svc := range mockServices {
		if req.Search != "" && !strings.Contains(strings.ToLower(svc.Name), strings.ToLower(req.Search)) {
			continue
		}
		if req.Category != "" && !strings.EqualFold(svc.Category, req.Category) {
			continue
		}
		response = append(response, structures.ServiceSummary{
			ID:     svc.ID,
			Name:   svc.Name,
			Status: aggregateStatus(svc.DAGs),
		})
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// GetService handles GET /services/{serviceId}.
//
// Request payload:  structures.GetServiceDetailRequestPayload  (path param)
// Response payload: structures.GetServiceDetailResponsePayload (JSON object)
//
// Returns 404 JSON error if serviceId is not found.
//
// Mock response (serviceId = "consumer"):
//
//	{
//	  "id":     "consumer",
//	  "name":   "Consumer",
//	  "status": "failed",
//	  "dags": [
//	    { "dag_id": "ecmwf_consumer",     "status": "healthy" },
//	    { "dag_id": "metoffice_consumer", "status": "failed"  },
//	    { "dag_id": "pvlive_consumer",    "status": "healthy" }
//	  ]
//	}
func GetService(w http.ResponseWriter, r *http.Request) {
	req := structures.GetServiceDetailRequestPayload{
		ServiceID: strings.TrimPrefix(r.URL.Path, "/services/"),
	}

	for _, svc := range mockServices {
		if svc.ID != req.ServiceID {
			continue
		}
		response := structures.GetServiceDetailResponsePayload{
			ID:     svc.ID,
			Name:   svc.Name,
			Status: aggregateStatus(svc.DAGs),
			DAGs:   svc.DAGs,
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(response)
		return
	}

	w.WriteHeader(http.StatusNotFound)
	json.NewEncoder(w).Encode(map[string]string{
		"error": "service not found: " + req.ServiceID,
	})
}
