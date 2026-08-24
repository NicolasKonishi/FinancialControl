package handlers

import (
	"net/http"
)

// HealthResponse is the JSON body returned by GET /health.
type HealthResponse struct {
	Status string `json:"status"`
}

// Health responds to health check requests.
func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}
