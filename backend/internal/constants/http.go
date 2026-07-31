package constants

import "time"

// HTTP header names.
const (
	HeaderContentType = "Content-Type"
	HeaderRequestID   = "X-Request-ID"
	ContentTypeJSON   = "application/json"
)

// DefaultHTTPTimeout is the timeout applied to every outbound HTTP request.
const DefaultHTTPTimeout = 10 * time.Second
