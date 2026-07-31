package constants

// Service and DAG health status values.
//
// Running and Queued exist because they are the common case against a live
// Airflow: several OCF DAGs are mid-run at any moment. Collapsing them into
// Unknown would report healthy services as broken.
const (
	StatusHealthy = "healthy"
	StatusFailed  = "failed"
	StatusRunning = "running"
	StatusQueued  = "queued"
	StatusUnknown = "unknown"
)
