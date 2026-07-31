// Package responses defines the outbound payload structures for each API endpoint.
package responses

import "time"

// DagRun represents an individual Airflow DAG run.
type DagRun struct {
	DagRunID    string  `json:"dag_run_id"`
	DAGID       string  `json:"dag_id"`
	LogicalDate string  `json:"logical_date"`
	StartDate   *string `json:"start_date"`
	EndDate     *string `json:"end_date"`
	State       string  `json:"state"`
	RunType     string  `json:"run_type"`
}

// RunSummary describes the most recent run of a DAG.
type RunSummary struct {
	RunID     string     `json:"run_id,omitempty"`
	State     string     `json:"state,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
}

// DAGDetail represents a single DAG inside a service detail response.
type DAGDetail struct {
	DAGID                 string         `json:"dag_id"`
	DAGDisplayName        string         `json:"dag_display_name"`
	IsPaused              bool           `json:"is_paused"`
	TimetableSummary      *string        `json:"timetable_summary"`
	NextDagrunLogicalDate *string        `json:"next_dagrun_logical_date"`
	Status                string         `json:"status"`
	AirflowURL            string         `json:"airflow_url,omitempty"`
	Schedule              string         `json:"schedule,omitempty"`
	LastRun               *RunSummary    `json:"last_run,omitempty"`
	Metrics               ServiceMetrics `json:"metrics"`
	Runs                  []DagRun       `json:"runs"`
}

// ServiceMetrics represents aggregated metrics for a service.
type ServiceMetrics struct {
	TotalRuns             int      `json:"total_runs"`
	FailedRuns            int      `json:"failed_runs"`
	SuccessRate           *float64 `json:"success_rate"`
	AvgDurationSeconds    *float64 `json:"avg_duration_seconds"`
	LatestDurationSeconds *float64 `json:"latest_duration_seconds"`
	LastRunAt             *string  `json:"last_run_at"`
	NextRunAt             *string  `json:"next_run_at"`
}

// ServiceResponse is the full service item for GET /services and GET /services/{id}.
type ServiceResponse struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Category  string         `json:"category"`
	Status    string         `json:"status"`
	DependsOn []string       `json:"depends_on"`
	DAGs      []DAGDetail    `json:"dags,omitempty"`
	Metrics   ServiceMetrics `json:"metrics"`
	Note      *string        `json:"note"`
}

// ServiceListResponse is the full response body for GET /services.
type ServiceListResponse []ServiceResponse

// ServiceDetailResponse is the full response body for GET /services/{serviceId}.
type ServiceDetailResponse = ServiceResponse
