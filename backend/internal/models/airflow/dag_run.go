package airflow

import "time"

// DagRun represents a single execution instance of an Airflow DAG.
type DagRun struct {
	DagRunID string `json:"dag_run_id"`
	DAGID    string `json:"dag_id"`
	State    string `json:"state"`
	RunType  string `json:"run_type"`

	LogicalDate   *time.Time `json:"logical_date"`
	ExecutionDate *time.Time `json:"execution_date"`
	StartDate     *time.Time `json:"start_date"`
	EndDate       *time.Time `json:"end_date"`
}

// DagRunList is the envelope returned by the dagRuns endpoints.
type DagRunList struct {
	DagRuns      []DagRun `json:"dag_runs"`
	TotalEntries int      `json:"total_entries"`
}
