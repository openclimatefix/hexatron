package routes

import (
	"net/http"

	"github.com/openclimatefix/hexatron/backend/constants"
	"github.com/openclimatefix/hexatron/backend/controllers"
)

// RegisterAirflowRoutes attaches the service/DAG health endpoints to mux.
//
//	GET /services      → controllers.ListServices
//	GET /services/{id} → controllers.GetService
func RegisterAirflowRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET "+constants.ServicesPath, controllers.ListServices)
	mux.HandleFunc("GET "+constants.ServiceByIDPath, controllers.GetService)
}
