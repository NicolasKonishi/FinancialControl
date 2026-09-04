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
	switch {
	case errors.Is(err, repository.ErrNotFound):
		http.Error(w, notFoundMessage, http.StatusNotFound)
	case errors.Is(err, repository.ErrWalletOwner):
		http.Error(w, "wallet does not belong to member", http.StatusBadRequest)
	case errors.Is(err, repository.ErrNoWallet):
		http.Error(w, "member has no wallet", http.StatusBadRequest)
	case errors.Is(err, repository.ErrNotCredit):
		http.Error(w, "wallet is not a credit card", http.StatusBadRequest)
	case errors.Is(err, repository.ErrInvalidAmount):
		http.Error(w, "invalid amount", http.StatusBadRequest)
	case errors.Is(err, repository.ErrInvoiceEmpty):
		http.Error(w, "invoice is already paid", http.StatusBadRequest)
	case errors.Is(err, repository.ErrStatementTypeMismatch):
		http.Error(w, err.Error(), http.StatusBadRequest)
	case errors.Is(err, repository.ErrStatementImported):
		http.Error(w, err.Error(), http.StatusConflict)
	default:
		log.Printf("store error: %v", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}
