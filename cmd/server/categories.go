package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

func categoriesHandler(store *categoryStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			listCategories(w, store)
		case http.MethodPost:
			createCategory(w, r, store)
		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func listCategories(w http.ResponseWriter, store *categoryStore) {
	categories := store.List()

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(categories); err != nil {
		log.Printf("failed to encode categories response: %v", err)
	}
}

func createCategory(w http.ResponseWriter, r *http.Request, store *categoryStore) {
	var input models.CreateCategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(input.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}

	category := store.Create(input)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(category); err != nil {
		log.Printf("failed to encode category response: %v", err)
	}
}
