// Package clients defines the data structures for Airflow client configuration.
package clients

// AirflowClientConfig holds configuration for the Airflow API client.
type AirflowClientConfig struct {
	BaseURL string
	Cookie  string
}

// The DAGRun placeholder that lived here has been replaced by
// models/airflow.DagRun, which matches the real Airflow API response.
