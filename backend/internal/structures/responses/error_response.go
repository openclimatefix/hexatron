package responses

// ErrorResponse is the standard JSON error body returned on failed requests.
//
//	{ "error": "service not found: consumer" }
type ErrorResponse struct {
	Error string `json:"error"`
}
