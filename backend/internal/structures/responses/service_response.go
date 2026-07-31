// Package responses defines the outbound payload structures for each API endpoint.
package responses

// ServiceSummary is a single item in the GET /services response.
//
//	{ "id": "site-forecast", "name": "Site Forecast", "status": "healthy" }
type ServiceSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// ServiceListResponse is the full response body for GET /services.
//
//	[
//	  { "id": "site-forecast", "name": "Site Forecast", "status": "healthy" },
//	  { "id": "consumer",      "name": "Consumer",      "status": "failed"  }
//	]
type ServiceListResponse []ServiceSummary

// DAGStatus is an individual DAG entry inside a service detail response.
//
//	{ "dag_id": "ecmwf_consumer", "status": "healthy" }
type DAGStatus struct {
	DAGID  string `json:"dag_id"`
	Status string `json:"status"`
}

// ServiceDetailResponse is the full response body for GET /services/{serviceId}.
//
//	{
//	  "id":     "consumer",
//	  "name":   "Consumer",
//	  "status": "failed",
//	  "dags":   [ ... ]
//	}
type ServiceDetailResponse struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Status string      `json:"status"`
	DAGs   []DAGStatus `json:"dags"`
}
