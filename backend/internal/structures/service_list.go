// Package structures defines the request and response payload shapes
// for the GET /services endpoint.
package structures

// GetServicesRequestPayload represents the optional query parameters
// accepted by GET /services.
//
//	?search=<string>   – filters services whose name contains the search term
//	?category=<string> – filters services by category (e.g. "Forecast")
type GetServicesRequestPayload struct {
	Search   string `form:"search"`
	Category string `form:"category"`
}

// ServiceSummary is a single entry in the GET /services response.
//
//	{
//	  "id":     "site-forecast",
//	  "name":   "Site Forecast",
//	  "status": "healthy"
//	}
type ServiceSummary struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Status string `json:"status"`
}

// GetServicesResponsePayload is the full response body for GET /services.
//
//	[
//	  { "id": "site-forecast", "name": "Site Forecast", "status": "healthy" },
//	  { "id": "consumer",      "name": "Consumer",       "status": "failed"  }
//	]
type GetServicesResponsePayload []ServiceSummary
