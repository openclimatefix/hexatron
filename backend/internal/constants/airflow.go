package constants

// Airflow environment variable keys and default values.
const (
	AirflowBaseURLEnv = "AIRFLOW_BASE_URL"
	AirflowCookieEnv  = "AIRFLOW_SESSION_COOKIE"
	AirflowDefaultURL = "http://127.0.0.1:38000"
)

// Airflow DAG run states returned by the Airflow REST API.
const (
	AirflowStateSuccess        = "success"
	AirflowStateFailed         = "failed"
	AirflowStateRunning        = "running"
	AirflowStateQueued         = "queued"
	AirflowStateUpstreamFailed = "upstream_failed"
)

// Airflow REST API paths.
const (
	AirflowDagsPath   = "/api/v1/dags"
	AirflowHealthPath = "/api/v1/health"
)

// AirflowSessionCookieName is the cookie Airflow's web session auth expects.
const AirflowSessionCookieName = "session"

const (
	// AirflowPageSize is the per-request page size for list endpoints. Airflow
	// caps this at 100 by default.
	AirflowPageSize = 100

	// AirflowMaxConcurrentRequests bounds the per-DAG fan-out so a large
	// services.yaml cannot flood Airflow's webserver.
	AirflowMaxConcurrentRequests = 8
)
