package routes

import (
	"net/http"

	"github.com/openclimatefix/hexatron/backend/internal/constants"
	"github.com/openclimatefix/hexatron/backend/internal/controllers"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
)

// RegisterAirflowRoutes attaches the service/DAG health endpoints to mux.
//
//	GET /services      → airflowController.ListServices
//	GET /services/{id} → airflowController.GetService
func RegisterAirflowRoutes(mux *http.ServeMux, cfg *configstructs.Config) {
	airflowController := controllers.NewAirflowController(cfg)
	mux.HandleFunc("GET "+constants.ServicesPath, airflowController.ListServices)
	mux.HandleFunc("GET "+constants.ServiceByIDPath, airflowController.GetService)
}
