// Package models holds the shared domain types used across all layers.
package models

import (
	"errors"
	"fmt"
	"path"
	"strings"
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

	// Planned marks a service that is intentionally not implemented yet, as
	// opposed to one Airflow simply has no data for. It only changes status
	// derivation when the service also has no DAGs — a planned service that
	// has since grown DAGs is judged on those like any other.
	Planned bool `yaml:"planned"`

	// NonCriticalDAGPatterns are globs over dag_ids the service claims but does
	// not depend on to function. A failure in one degrades the service rather
	// than downing it.
	//
	// Without this, one failing input reds out a whole service: cloudcasting
	// feeds the solar forecast, but a cloudcasting failure does not mean the
	// solar forecast is down. Patterns here must also match a dag_patterns
	// entry — this narrows severity, it does not add membership.
	NonCriticalDAGPatterns []string `yaml:"non_critical_dag_patterns"`

	// HealthCheck is an out-of-band liveness probe, for services Airflow cannot
	// speak for. The UI and the API are not DAGs, so without one they can only
	// ever report "unknown" however healthy they actually are.
	HealthCheck *HealthCheck `yaml:"health_check"`
}

// HealthCheck describes how to probe a service directly.
//
// Two transports, because the fleet speaks two protocols: `http` covers the
// REST surfaces (UI, API), `grpc` covers servers implementing the standard
// grpc.health.v1.Health service.
type HealthCheck struct {
	// Type is "http" or "grpc". Empty defaults to "http".
	Type string `yaml:"type"`

	// Target is a URL for http, or a host:port dial address for grpc.
	Target string `yaml:"target"`

	// ExpectStatus is the HTTP status that counts as healthy. Defaults to 200.
	// Ignored by grpc probes.
	ExpectStatus int `yaml:"expect_status"`

	// ExpectBodyContains is a string the healthy response must contain.
	//
	// A status code alone is not enough when the target is a rendered page: an
	// app that catches its own backend failure and renders an error state still
	// returns 200, so a code-only probe reports healthy while the page is
	// visibly broken. Empty skips the check. Ignored by grpc probes.
	ExpectBodyContains string `yaml:"expect_body_contains"`

	// GRPCService names a single registered service to check. Empty asks about
	// the server as a whole, which is what most deployments answer for.
	GRPCService string `yaml:"grpc_service"`

	// TimeoutSeconds bounds one probe. Defaults to 3.
	TimeoutSeconds int `yaml:"timeout_seconds"`
}

// Probe transports.
const (
	HealthCheckHTTP = "http"
	HealthCheckGRPC = "grpc"
)

// Normalised returns the check with defaults applied, so callers never have to
// second-guess a partially specified YAML block.
func (h HealthCheck) Normalised() HealthCheck {
	if h.Type == "" {
		h.Type = HealthCheckHTTP
	}
	if h.ExpectStatus == 0 {
		h.ExpectStatus = 200
	}
	if h.TimeoutSeconds == 0 {
		h.TimeoutSeconds = 3
	}
	return h
}

// ValidateHealthCheck rejects a check that could never succeed, at load time
// rather than on the first request.
func ValidateHealthCheck(h HealthCheck) error {
	n := h.Normalised()
	if n.Target == "" {
		return errNoTarget
	}
	switch n.Type {
	case HealthCheckHTTP:
		if !strings.HasPrefix(n.Target, "http://") && !strings.HasPrefix(n.Target, "https://") {
			return fmt.Errorf("http target %q must start with http:// or https://", n.Target)
		}
	case HealthCheckGRPC:
		// A dial address, not a URL — reject the scheme people reach for first.
		if strings.Contains(n.Target, "://") {
			return fmt.Errorf("grpc target %q must be host:port, not a URL", n.Target)
		}
		if !strings.Contains(n.Target, ":") {
			return fmt.Errorf("grpc target %q must include a port", n.Target)
		}
	default:
		return fmt.Errorf("unknown health_check type %q (want %q or %q)",
			n.Type, HealthCheckHTTP, HealthCheckGRPC)
	}
	return nil
}

var errNoTarget = errors.New("health_check has no target")

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

// IsCritical reports whether a failure in dagID should down the service.
// DAGs the service does not claim at all are treated as critical, since the
// question only arises for ones it does.
func (s Service) IsCritical(dagID string) bool {
	for _, pattern := range s.NonCriticalDAGPatterns {
		if matchesDAGPattern(pattern, dagID) {
			return false
		}
	}
	return true
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
