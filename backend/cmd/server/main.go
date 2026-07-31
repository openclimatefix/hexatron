package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/openclimatefix/hexatron/backend/internal/config"
	"github.com/openclimatefix/hexatron/backend/internal/routes"
	"github.com/openclimatefix/hexatron/backend/internal/services"
	configstructs "github.com/openclimatefix/hexatron/backend/internal/structures/config"
)

// startupTimeout bounds the startup config check.
const startupTimeout = 30 * time.Second

func main() {
	cfg := config.Load()
	router := routes.NewRouter(cfg)

	logConfigDrift(cfg)

	fmt.Printf("Hexatron backend listening on http://localhost%s\n", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

// logConfigDrift warns when services.yaml and Airflow disagree.
func logConfigDrift(cfg *configstructs.Config) {
	ctx, cancel := context.WithTimeout(context.Background(), startupTimeout)
	defer cancel()

	unmatched, unclaimed, err := services.NewAirflowService(cfg).ConfigDrift(ctx)
	if err != nil {
		log.Printf("warning: could not check services.yaml against airflow: %v", err)
		return
	}

	if len(unmatched) > 0 {
		log.Printf("warning: %d DAG pattern(s) matched no airflow DAG: %s",
			len(unmatched), strings.Join(unmatched, ", "))
	}
	if len(unclaimed) > 0 {
		log.Printf("note: %d airflow DAG(s) not claimed by any service: %s",
			len(unclaimed), strings.Join(unclaimed, ", "))
	}
}
