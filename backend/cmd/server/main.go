package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/openclimatefix/hexatron/backend/config"
	"github.com/openclimatefix/hexatron/backend/routes"
)

func main() {
	cfg := config.Load()
	router := routes.NewRouter()

	fmt.Printf("Hexatron backend listening on http://localhost%s\n", cfg.Addr)
	if err := http.ListenAndServe(cfg.Addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
