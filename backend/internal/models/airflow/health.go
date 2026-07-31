package airflow

// Health represents the Airflow GET /api/v1/health response.
type Health struct {
	Metadatabase HealthComponent `json:"metadatabase"`
	Scheduler    HealthComponent `json:"scheduler"`
}

// HealthComponent represents an individual Airflow component's health status.
type HealthComponent struct {
	Status string `json:"status"`
}
