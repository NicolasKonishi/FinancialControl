package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"

	"github.com/NicolasKonishi/FinancialControl/internal/repository"
)

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		log.Printf("failed to encode JSON response: %v", err)
	}
}

func parsePositiveID(w http.ResponseWriter, raw string) (int, bool) {
	id, err := strconv.Atoi(raw)
	if err != nil || id < 1 {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return 0, false
	}
	return id, true
}

func writeStoreError(w http.ResponseWriter, err error, notFoundMessage string) {
	if errors.Is(err, repository.ErrNotFound) {
		http.Error(w, notFoundMessage, http.StatusNotFound)
		return
	}
	log.Printf("store error: %v", err)
	http.Error(w, "internal server error", http.StatusInternalServerError)
}
