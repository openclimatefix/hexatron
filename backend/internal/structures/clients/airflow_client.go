// Package clients defines the data structures for Airflow client configuration.
package clients

// AirflowClientConfig holds configuration for the Airflow API client.
type AirflowClientConfig struct {
	BaseURL string
	Cookie  string
}
