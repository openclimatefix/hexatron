// Package structures defines the request and response payload shapes
// for the GET /services/{serviceId} endpoint.
package structures

// GetServiceDetailRequestPayload represents the path parameter accepted by
// GET /services/{serviceId}. There is no request body.
//
//	Path: /services/{serviceId}
type GetServiceDetailRequestPayload struct {
	ServiceID string `path:"serviceId"`
}

// DAGStatus is an individual DAG entry inside a service detail response.
//
//	{
//	  "dag_id": "ecmwf_consumer",
//	  "status": "healthy"
//	}
type DAGStatus struct {
	DAGID  string `json:"dag_id"`
	Status string `json:"status"`
}

// GetServiceDetailResponsePayload is the full response body for
// GET /services/{serviceId}.
//
//	{
//	  "id":     "consumer",
//	  "name":   "Consumer",
//	  "status": "failed",
//	  "dags": [
//	    { "dag_id": "ecmwf_consumer",     "status": "healthy" },
//	    { "dag_id": "metoffice_consumer", "status": "failed"  }
//	  ]
//	}
type GetServiceDetailResponsePayload struct {
	ID     string      `json:"id"`
	Name   string      `json:"name"`
	Status string      `json:"status"`
	DAGs   []DAGStatus `json:"dags"`
}
