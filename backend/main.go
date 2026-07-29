package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/openclimatefix/hexatron/backend/internal/api"
)

func main() {
	router := api.NewRouter()

	addr := ":8080"
	fmt.Printf("Hexatron backend listening on http://localhost%s\n", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
