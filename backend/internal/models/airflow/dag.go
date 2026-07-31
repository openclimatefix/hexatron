// Package airflow holds Airflow domain models used in Phase 2.
package airflow

// DAG represents an Airflow DAG definition returned by the REST API.
type DAG struct {
	DAGID       string `json:"dag_id"`
	IsPaused    bool   `json:"is_paused"`
	IsActive    bool   `json:"is_active"`
	Description string `json:"description"`
}
