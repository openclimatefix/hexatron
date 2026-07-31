package constants

// Airflow environment variable keys and default values.
const (
	AirflowBaseURLEnv = "AIRFLOW_BASE_URL"
	AirflowCookieEnv  = "AIRFLOW_SESSION_COOKIE"
	AirflowDefaultURL = "http://127.0.0.1:38000"
)

// Airflow DAG run states returned by the Airflow REST API.
const (
	AirflowStateSuccess = "success"
	AirflowStateFailed  = "failed"
	AirflowStateRunning = "running"
	AirflowStateQueued  = "queued"
)
