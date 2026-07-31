package airflow

// TaskInstance represents a single task within an Airflow DAG run.
type TaskInstance struct {
	TaskID    string `json:"task_id"`
	DAGID     string `json:"dag_id"`
	DagRunID  string `json:"dag_run_id"`
	State     string `json:"state"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}
