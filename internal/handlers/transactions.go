package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
	"github.com/NicolasKonishi/FinancialControl/internal/repository"
)

// TransactionStore is the persistence dependency for transaction handlers.
type TransactionStore interface {
	CreateTransaction(ctx context.Context, tx models.Transaction) (models.Transaction, error)
	ListTransactions(ctx context.Context) ([]models.Transaction, error)
	GetTransactionByID(ctx context.Context, id int) (models.Transaction, error)
	UpdateTransaction(ctx context.Context, id int, tx models.Transaction) (models.Transaction, error)
	DeleteTransaction(ctx context.Context, id int) error
}

// Transactions handles transaction HTTP endpoints.
type Transactions struct {
	Store      TransactionStore
	Categories CategoryStore
	Members    MemberStore
}

// ListOrCreate handles GET and POST /transactions.
func (h *Transactions) ListOrCreate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.Store.ListTransactions(r.Context())
		if err != nil {
			writeStoreError(w, err, "transaction not found")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		h.create(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Transactions) create(w http.ResponseWriter, r *http.Request) {
	var input models.CreateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	tx, ok := h.buildFromInput(w, r, input.CategoryID, input.MemberID, input.WalletID, input.Type, input.Description, input.Amount, input.Date)
	if !ok {
		return
	}

	created, err := h.Store.CreateTransaction(r.Context(), tx)
	if err != nil {
		if errors.Is(err, repository.ErrWalletOwner) {
			http.Error(w, "wallet does not belong to this person", http.StatusBadRequest)
			return
		}
		writeStoreError(w, err, "transaction not found")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// GetByID handles GET /transactions/{id}.
func (h *Transactions) GetByID(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	tx, err := h.Store.GetTransactionByID(r.Context(), id)
	if err != nil {
		writeStoreError(w, err, "transaction not found")
		return
	}
	writeJSON(w, http.StatusOK, tx)
}

// Update handles PUT /transactions/{id}.
func (h *Transactions) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var input models.UpdateTransactionInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	tx, ok := h.buildFromInput(w, r, input.CategoryID, input.MemberID, input.WalletID, input.Type, input.Description, input.Amount, input.Date)
	if !ok {
		return
	}

	updated, err := h.Store.UpdateTransaction(r.Context(), id, tx)
	if err != nil {
		if errors.Is(err, repository.ErrWalletOwner) {
			http.Error(w, "wallet does not belong to this person", http.StatusBadRequest)
			return
		}
		writeStoreError(w, err, "transaction not found")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /transactions/{id}.
func (h *Transactions) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	if err := h.Store.DeleteTransaction(r.Context(), id); err != nil {
		writeStoreError(w, err, "transaction not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Transactions) buildFromInput(
	w http.ResponseWriter,
	r *http.Request,
	categoryID int,
	memberID *int,
	walletID *int,
	txType, description string,
	amount float64,
	dateStr string,
) (models.Transaction, bool) {
	txType = strings.ToLower(strings.TrimSpace(txType))
	if txType != models.TransactionTypeIncome && txType != models.TransactionTypeExpense {
		http.Error(w, "type must be income or expense", http.StatusBadRequest)
		return models.Transaction{}, false
	}
	if amount <= 0 {
		http.Error(w, "amount must be greater than zero", http.StatusBadRequest)
		return models.Transaction{}, false
	}
	if strings.TrimSpace(description) == "" {
		http.Error(w, "description is required", http.StatusBadRequest)
		return models.Transaction{}, false
	}

	if _, err := h.Categories.GetCategoryByID(r.Context(), categoryID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "category not found", http.StatusBadRequest)
			return models.Transaction{}, false
		}
		writeStoreError(w, err, "category not found")
		return models.Transaction{}, false
	}

	if memberID != nil {
		if _, err := h.Members.GetMemberByID(r.Context(), *memberID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, "member not found", http.StatusBadRequest)
				return models.Transaction{}, false
			}
			writeStoreError(w, err, "member not found")
			return models.Transaction{}, false
		}
	}

	date, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		http.Error(w, "date must be YYYY-MM-DD", http.StatusBadRequest)
		return models.Transaction{}, false
	}

	return models.Transaction{
		CategoryID:  categoryID,
		MemberID:    memberID,
		WalletID:    walletID,
		Type:        txType,
		Description: strings.TrimSpace(description),
		Amount:      amount,
		Date:        date.UTC(),
	}, true
}
