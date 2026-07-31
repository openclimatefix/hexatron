// Package models holds the shared domain types used across all layers.
package models

import (
	"path"
	"time"
)

// Service represents a business service defined in data/services.yaml.
type Service struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Category string `yaml:"category"`

	// DependsOn lists upstream service ids, for the dashboard's dependency graph.
	DependsOn []string `yaml:"depends_on"`

	// DAGPatterns are globs over dag_id; empty means the service has no DAGs yet.
	DAGPatterns []string `yaml:"dag_patterns"`
}

// MatchDAGs returns the dag ids this service claims, in the order given, deduplicated.
func (s Service) MatchDAGs(dagIDs []string) []string {
	claimed := make([]string, 0, len(dagIDs))
	for _, dagID := range dagIDs {
		for _, pattern := range s.DAGPatterns {
			if matchesDAGPattern(pattern, dagID) {
				claimed = append(claimed, dagID)
				break
			}
		}
	}
	return claimed
}

// MatchDAGPattern returns the dag ids a single pattern claims.
func MatchDAGPattern(pattern string, dagIDs []string) []string {
	matched := make([]string, 0, len(dagIDs))
	for _, dagID := range dagIDs {
		if matchesDAGPattern(pattern, dagID) {
			matched = append(matched, dagID)
		}
	}
	return matched
}

// ValidateDAGPattern reports whether a pattern is a well-formed glob.
func ValidateDAGPattern(pattern string) error {
	_, err := path.Match(pattern, "")
	return err
}

// matchesDAGPattern reports whether the glob pattern claims dagID.
func matchesDAGPattern(pattern, dagID string) bool {
	matched, err := path.Match(pattern, dagID)
	return err == nil && matched
}

// DAGStatus represents the computed runtime status of a single DAG.
type DAGStatus struct {
	DAGID  string
	Name   string
	Status string

	// IsPaused reports that scheduling is off; Schedule is the DAG's cron expression.
	IsPaused bool
	Schedule string

	// LastRun describes the most recent run, nil if the DAG has never run.
	LastRun *RunSummary

	// AirflowURL deep-links to the DAG in Airflow.
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

// ServiceDetail represents the full status detail of a service, including its DAGs.
type ServiceDetail struct {
	ID       string
	Name     string
	Category string
	Status   string
	DAGs     []DAGStatus
}
