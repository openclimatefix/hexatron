// Package requests defines the inbound payload structures for each API endpoint.
package requests

// GetServicesRequestPayload captures the optional query parameters for
// GET /services.
//
//	?search=<string>   – case-insensitive name filter
//	?category=<string> – exact category match
type GetServicesRequestPayload struct {
	Search   string `form:"search"`
	Category string `form:"category"`
}

// GetServiceDetailRequestPayload captures the path parameter for
// GET /services/{serviceId}. There is no request body.
type GetServiceDetailRequestPayload struct {
	ServiceID string `path:"serviceId"`
}
