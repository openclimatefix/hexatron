// Package models holds the shared domain types used across all layers.
package models

import "time"

// Service represents a business service defined in data/services.yaml.
type Service struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Category string   `yaml:"category"`
	DAGIDs   []string `yaml:"dags"`
}

// DAGStatus represents the computed runtime status of a single DAG, along with
// the context needed to act on it: whether scheduling is paused, when it last
// ran, and where to open it in Airflow.
type DAGStatus struct {
	DAGID  string
	Name   string
	Status string

	// IsPaused reports that scheduling is switched off, so Status reflects a run
	// that may be old. Schedule is the DAG's cron expression.
	IsPaused bool
	Schedule string

	// LastRun describes the most recent run, nil if the DAG has never run.
	LastRun *RunSummary

	// AirflowURL deep-links to the DAG in Airflow, which stays the source of
	// truth for logs and task failures.
	AirflowURL string
}

// RunSummary is the part of an Airflow DAG run Hexatron surfaces.
type RunSummary struct {
	RunID     string
	State     string
	StartDate *time.Time
	EndDate   *time.Time
}

// ServiceSummary represents the computed status summary of a service.
type ServiceSummary struct {
	ID       string
	Name     string
	Category string
	Status   string
}

// ServiceDetail represents the full computed status detail of a service,
// including individual DAG statuses.
type ServiceDetail struct {
	ID       string
	Name     string
	Category string
	Status   string
	DAGs     []DAGStatus
}
