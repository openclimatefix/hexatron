package airflow

// DagRun represents a single execution instance of an Airflow DAG.
type DagRun struct {
	DagRunID      string `json:"dag_run_id"`
	DAGID         string `json:"dag_id"`
	State         string `json:"state"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
	ExecutionDate string `json:"execution_date"`
}
