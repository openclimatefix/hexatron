// Package models holds the shared domain types used across all layers.
package models

// Service represents a business service defined in data/services.yaml.
type Service struct {
	ID       string   `yaml:"id"`
	Name     string   `yaml:"name"`
	Category string   `yaml:"category"`
	DAGIDs   []string `yaml:"dags"`
}

// DAGStatus represents the computed runtime status of a single DAG.
type DAGStatus struct {
	DAGID  string
	Status string
}

// ServiceSummary represents the computed status summary of a service.
type ServiceSummary struct {
	ID     string
	Name   string
	Status string
}

// ServiceDetail represents the full computed status detail of a service,
// including individual DAG statuses.
type ServiceDetail struct {
	ID     string
	Name   string
	Status string
	DAGs   []DAGStatus
}

