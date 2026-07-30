// Package mock defines mock DAG health statuses used prior to Phase 2 Airflow REST API integration.
package mock

import "github.com/openclimatefix/hexatron/backend/constants"

// mockDAGOverrides holds explicit DAG status overrides for testing (e.g. failing DAGs).
var mockDAGOverrides = map[string]string{
	"metoffice_consumer": constants.StatusFailed,
}

// GetDAGStatus returns the mock health status for a given DAG ID.
// If the DAG is not in the override map, it defaults to "healthy".
func GetDAGStatus(dagID string) string {
	if status, exists := mockDAGOverrides[dagID]; exists {
		return status
	}
	return constants.StatusHealthy
}
