package constants

// User-facing error and info messages.
const (
	ErrServiceNotFound = "service not found"

	// Airflow upstream failures, shown instead of marking every service failed.
	ErrAirflowUnauthorized = "airflow session cookie is missing or expired"
	ErrAirflowUnreachable  = "could not reach airflow"
)
