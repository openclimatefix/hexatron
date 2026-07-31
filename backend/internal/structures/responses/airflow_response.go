// Package responses includes the raw Airflow API response shapes.
// These are used in Phase 2 when live Airflow integration is implemented.
package responses

// AirflowHealthResponse mirrors the Airflow GET /api/v1/health response.
type AirflowHealthResponse struct {
	Metadatabase AirflowComponent `json:"metadatabase"`
	Scheduler    AirflowComponent `json:"scheduler"`
}

// AirflowComponent represents an Airflow component's health status.
type AirflowComponent struct {
	Status string `json:"status"`
}
