// Package clients defines the data structures for Airflow client configuration.
package clients

// AirflowClientConfig holds configuration for the Airflow API client.
type AirflowClientConfig struct {
	BaseURL string
	Cookie  string
}

// TODO : CHECK API Respone and match this struct fields.
// DAGRun represents a single DAG run object returned by the Airflow API.
type DAGRun struct {
	DAGRunID string `json:"dag_run_id"`
	State    string `json:"state"`
}

