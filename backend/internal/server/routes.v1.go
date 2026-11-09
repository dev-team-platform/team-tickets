package server

import (
	"encoding/json"
	"net/http"
)

type healthResponse struct {
	Status string `json:"status"`
}

// Register public endpoints on public and all other endpoints on protected.
func registerV1(public, protected *http.ServeMux) {
	public.HandleFunc("GET /api/v1/health/live", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(healthResponse{Status: "ok"})
	})
}
