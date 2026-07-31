package constants

// User-facing error and info messages.
const (
	ErrServiceNotFound = "service not found"

	// Airflow upstream failures. The dashboard shows these instead of painting
	// every service red, so a connectivity problem is not mistaken for an
	// outage in the services themselves.
	ErrAirflowUnauthorized = "airflow session cookie is missing or expired"
	ErrAirflowUnreachable  = "could not reach airflow"
)
