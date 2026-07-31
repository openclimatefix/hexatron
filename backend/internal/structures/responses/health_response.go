package responses

// HealthResponse is the response body for GET /health.
//
//	{ "status": "healthy" }
type HealthResponse struct {
	Status string `json:"status"`
}
