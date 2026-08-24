package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// MemberStore is the persistence dependency for member handlers.
type MemberStore interface {
	CreateMember(ctx context.Context, input models.CreateMemberInput) (models.Member, error)
	ListMembers(ctx context.Context) ([]models.Member, error)
	GetMemberByID(ctx context.Context, id int) (models.Member, error)
	UpdateMember(ctx context.Context, id int, input models.UpdateMemberInput) (models.Member, error)
	DeleteMember(ctx context.Context, id int) error
}

// Members handles family member HTTP endpoints.
type Members struct {
	Store MemberStore
}

// ListOrCreate handles GET and POST /members.
func (h *Members) ListOrCreate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.Store.ListMembers(r.Context())
		if err != nil {
			writeStoreError(w, err, "member not found")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		h.create(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Members) create(w http.ResponseWriter, r *http.Request) {
	var input models.CreateMemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if input.MonthlySalary < 0 {
		http.Error(w, "monthly_salary must be >= 0", http.StatusBadRequest)
		return
	}

	member, err := h.Store.CreateMember(r.Context(), input)
	if err != nil {
		writeStoreError(w, err, "member not found")
		return
	}
	writeJSON(w, http.StatusCreated, member)
}

// Update handles PUT /members/{id}.
func (h *Members) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var input models.UpdateMemberInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(input.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if input.MonthlySalary < 0 {
		http.Error(w, "monthly_salary must be >= 0", http.StatusBadRequest)
		return
	}

	member, err := h.Store.UpdateMember(r.Context(), id, input)
	if err != nil {
		writeStoreError(w, err, "member not found")
		return
	}
	writeJSON(w, http.StatusOK, member)
}

// Delete handles DELETE /members/{id}.
func (h *Members) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}
	if err := h.Store.DeleteMember(r.Context(), id); err != nil {
		writeStoreError(w, err, "member not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
