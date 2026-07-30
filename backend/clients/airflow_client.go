// Package clients is the only component aware of Airflow APIs.
//
// Planned methods: GetHealth, GetLatestDagRun, GetTaskInstances.
//
// Phase 2.
package clients

import (
	"errors"
	"net/http"

	"github.com/openclimatefix/hexatron/backend/constants"
	clientstructs "github.com/openclimatefix/hexatron/backend/structures/clients"
)

// AirflowClient communicates with the Airflow REST API.
type AirflowClient struct {
	config     clientstructs.AirflowClientConfig
	httpClient *http.Client
}

// NewAirflowClient returns a new AirflowClient using the given config.
func NewAirflowClient(cfg clientstructs.AirflowClientConfig) *AirflowClient {
	return &AirflowClient{
		config:     cfg,
		httpClient: NewHTTPClient(),
	}
}

// TODO: GetHealth calls GET /api/v1/health on the Airflow REST API.
func (c *AirflowClient) GetHealth() error {
	return nil
}

// TODO: GetLatestDagRun fetches the most recent run for the given DAG ID
// Here we have to Do API Calls.
func (c *AirflowClient) GetLatestDagRun(dagID string) (*clientstructs.DAGRun, error) {
	if dagID == "metoffice_consumer" {
		return nil, errors.New(constants.AirflowStateFailed)
	}
	return &clientstructs.DAGRun{
		DAGRunID: "latest",
		State:    constants.AirflowStateSuccess,
	}, nil
}

// TODO: GetTaskInstances fetches task instances for a specific DAG run.
func (c *AirflowClient) GetTaskInstances(dagID, dagRunID string) error {
	return nil
}
