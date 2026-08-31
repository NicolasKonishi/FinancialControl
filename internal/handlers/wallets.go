package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
	"github.com/NicolasKonishi/FinancialControl/internal/repository"
)

// WalletStore is the persistence dependency for wallet handlers.
type WalletStore interface {
	CreateWallet(ctx context.Context, input models.CreateWalletInput) (models.Wallet, error)
	ListWallets(ctx context.Context) ([]models.Wallet, error)
	GetWalletByID(ctx context.Context, id int) (models.Wallet, error)
	UpdateWallet(ctx context.Context, id int, input models.UpdateWalletInput) (models.Wallet, error)
	DeleteWallet(ctx context.Context, id int) error
}

// Wallets handles account/box balance HTTP endpoints.
type Wallets struct {
	Store   WalletStore
	Members MemberStore
}

// ListOrCreate handles GET and POST /wallets.
func (h *Wallets) ListOrCreate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.Store.ListWallets(r.Context())
		if err != nil {
			writeStoreError(w, err, "wallet not found")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		h.create(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Wallets) create(w http.ResponseWriter, r *http.Request) {
	var input models.CreateWalletInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	normalized, ok := h.validate(w, r, input)
	if !ok {
		return
	}
	created, err := h.Store.CreateWallet(r.Context(), normalized)
	if err != nil {
		writeStoreError(w, err, "wallet not found")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Update handles PUT /wallets/{id}.
func (h *Wallets) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}
	var input models.UpdateWalletInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	normalized, ok := h.validate(w, r, models.CreateWalletInput(input))
	if !ok {
		return
	}
	updated, err := h.Store.UpdateWallet(r.Context(), id, models.UpdateWalletInput(normalized))
	if err != nil {
		writeStoreError(w, err, "wallet not found")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /wallets/{id}.
func (h *Wallets) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}
	if err := h.Store.DeleteWallet(r.Context(), id); err != nil {
		writeStoreError(w, err, "wallet not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Wallets) validate(w http.ResponseWriter, r *http.Request, input models.CreateWalletInput) (models.CreateWalletInput, bool) {
	if strings.TrimSpace(input.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return models.CreateWalletInput{}, false
	}
	kind := models.NormalizeWalletKind(strings.ToLower(strings.TrimSpace(input.Kind)))
	if !models.ValidWalletKind(kind) {
		http.Error(w, "kind must be checking, savings, benefit, company, or credit", http.StatusBadRequest)
		return models.CreateWalletInput{}, false
	}
	if input.MemberID != nil {
		if _, err := h.Members.GetMemberByID(r.Context(), *input.MemberID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, "member not found", http.StatusBadRequest)
				return models.CreateWalletInput{}, false
			}
			writeStoreError(w, err, "member not found")
			return models.CreateWalletInput{}, false
		}
	}
	return models.CreateWalletInput{
		Name:     strings.TrimSpace(input.Name),
		Kind:     kind,
		MemberID: input.MemberID,
		Balance:  input.Balance,
	}, true
}
