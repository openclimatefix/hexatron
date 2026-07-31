// Package models holds the shared domain types used across all layers.
package models

import (
	"path"
	"time"
)

// Service represents a business service defined in data/services.yaml.
//
// A service does not name its DAGs. It carries glob patterns matched against
// the dag_ids Airflow reports, so the set of DAGs behind a service follows
// Airflow rather than this config drifting out of date behind it.
type Service struct {
	ID       string `yaml:"id"`
	Name     string `yaml:"name"`
	Category string `yaml:"category"`

	// DependsOn lists upstream service ids, for the dashboard's dependency graph.
	DependsOn []string `yaml:"depends_on"`

	// DAGPatterns are globs over dag_id. Empty means the service has no DAGs in
	// Airflow yet, which is a legitimate state, not a misconfiguration.
	DAGPatterns []string `yaml:"dag_patterns"`
}

// MatchDAGs returns the dag ids this service claims out of every dag id Airflow
// reported, in the order given. A DAG matching several patterns is returned
// once; services may legitimately claim the same DAG as each other.
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

// MatchDAGPattern returns the dag ids a single pattern claims. A pattern that
// claims nothing is the DAG-membership equivalent of a typo'd dag id, which is
// why config drift reports on it separately.
func MatchDAGPattern(pattern string, dagIDs []string) []string {
	matched := make([]string, 0, len(dagIDs))
	for _, dagID := range dagIDs {
		if matchesDAGPattern(pattern, dagID) {
			matched = append(matched, dagID)
		}
	}
	return matched
}

// ValidateDAGPattern reports whether a pattern is a well-formed glob. A
// malformed one never matches, so the registry rejects it at load rather than
// letting a service silently lose its DAGs.
func ValidateDAGPattern(pattern string) error {
	_, err := path.Match(pattern, "")
	return err
}

// matchesDAGPattern reports whether pattern claims dagID. Patterns are globs:
// "uk-forecast-*" claims uk-forecast-site, and because * does not span the
// whole id, "nl-forecast" does not claim nl-consume-ned-nl-forecast.
func matchesDAGPattern(pattern, dagID string) bool {
	matched, err := path.Match(pattern, dagID)
	return err == nil && matched
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
