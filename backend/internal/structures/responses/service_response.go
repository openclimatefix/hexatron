// Package responses defines the outbound payload structures for each API endpoint.
package responses

import "time"

// ServiceSummary is a single item in the GET /services response.
type ServiceSummary struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Category string `json:"category,omitempty"`
	Status   string `json:"status"`
}

// ServiceListResponse is the full response body for GET /services.
type ServiceListResponse []ServiceSummary

// DAGStatus is an individual DAG entry inside a service detail response.
type DAGStatus struct {
	DAGID  string `json:"dag_id"`
	Name   string `json:"name,omitempty"`
	Status string `json:"status"`

	IsPaused bool   `json:"is_paused"`
	Schedule string `json:"schedule,omitempty"`

	LastRun    *RunSummary `json:"last_run,omitempty"`
	AirflowURL string      `json:"airflow_url,omitempty"`
}

// RunSummary describes the most recent run of a DAG.
type RunSummary struct {
	RunID     string     `json:"run_id"`
	State     string     `json:"state"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

// ServiceDetailResponse is the full response body for GET /services/{serviceId}.
type ServiceDetailResponse struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Category string      `json:"category,omitempty"`
	Status   string      `json:"status"`
	DAGs     []DAGStatus `json:"dags"`
}
