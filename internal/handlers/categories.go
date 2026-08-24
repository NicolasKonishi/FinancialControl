package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// CategoryStore is the persistence dependency for category handlers.
type CategoryStore interface {
	CreateCategory(ctx context.Context, input models.CreateCategoryInput) (models.Category, error)
	ListCategories(ctx context.Context) ([]models.Category, error)
	GetCategoryByID(ctx context.Context, id int) (models.Category, error)
	UpdateCategory(ctx context.Context, id int, input models.UpdateCategoryInput) (models.Category, error)
	DeleteCategory(ctx context.Context, id int) error
}

// Categories handles collection routes for /categories.
type Categories struct {
	Store CategoryStore
}

// ListOrCreate handles GET and POST /categories.
func (h *Categories) ListOrCreate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		h.list(w, r)
	case http.MethodPost:
		h.create(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Categories) list(w http.ResponseWriter, r *http.Request) {
	categories, err := h.Store.ListCategories(r.Context())
	if err != nil {
		writeStoreError(w, err, "category not found")
		return
	}
	writeJSON(w, http.StatusOK, categories)
}

func (h *Categories) create(w http.ResponseWriter, r *http.Request) {
	var input models.CreateCategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if input.Icon == "" {
		input.Icon = "other"
	}

	category, err := h.Store.CreateCategory(r.Context(), input)
	if err != nil {
		writeStoreError(w, err, "category not found")
		return
	}
	writeJSON(w, http.StatusCreated, category)
}

// GetByID handles GET /categories/{id}.
func (h *Categories) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	category, err := h.Store.GetCategoryByID(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "category not found")
		return
	}
	writeJSON(w, http.StatusOK, category)
}

// Update handles PUT /categories/{id}.
func (h *Categories) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var input models.UpdateCategoryInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if input.Icon == "" {
		input.Icon = "other"
	}

	category, err := h.Store.UpdateCategory(r.Context(), id, input)
	if err != nil {
		writeStoreError(w, err, "category not found")
		return
	}
	writeJSON(w, http.StatusOK, category)
}

// Delete handles DELETE /categories/{id}.
func (h *Categories) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	if err := h.Store.DeleteCategory(r.Context(), id); err != nil {
		writeStoreError(w, err, "category not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
