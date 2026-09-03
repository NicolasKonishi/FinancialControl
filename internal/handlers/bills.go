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

// BillStore is the persistence dependency for bill handlers.
type BillStore interface {
	CreateBill(ctx context.Context, bill models.Bill) (models.Bill, error)
	ListBills(ctx context.Context) ([]models.Bill, error)
	GetBillByID(ctx context.Context, id int) (models.Bill, error)
	UpdateBill(ctx context.Context, id int, bill models.Bill) (models.Bill, error)
	DeleteBill(ctx context.Context, id int) error
	ListBillPayments(ctx context.Context, year, month int) ([]models.BillPayment, error)
	SetBillPaid(ctx context.Context, billID, year, month int, paid bool, paidByMemberID, walletID *int) error
}

// Bills handles monthly bill HTTP endpoints.
type Bills struct {
	Store      BillStore
	Categories CategoryStore
	Members    MemberStore
	Wallets    interface {
		ListWallets(ctx context.Context) ([]models.Wallet, error)
	}
}

// ListOrCreate handles GET and POST /bills.
func (h *Bills) ListOrCreate(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		items, err := h.Store.ListBills(r.Context())
		if err != nil {
			writeStoreError(w, err, "bill not found")
			return
		}
		writeJSON(w, http.StatusOK, items)
	case http.MethodPost:
		h.create(w, r)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (h *Bills) create(w http.ResponseWriter, r *http.Request) {
	var input models.CreateBillInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	bill, ok := h.buildFromInput(w, r, input)
	if !ok {
		return
	}

	created, err := h.Store.CreateBill(r.Context(), bill)
	if err != nil {
		writeStoreError(w, err, "bill not found")
		return
	}
	writeJSON(w, http.StatusCreated, created)
}

// Update handles PUT /bills/{id}.
func (h *Bills) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var input models.UpdateBillInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	bill, ok := h.buildFromInput(w, r, models.CreateBillInput{
		Name:         input.Name,
		Amount:       input.Amount,
		AmountMode:   input.AmountMode,
		InterestRate: input.InterestRate,
		CategoryID:   input.CategoryID,
		MemberIDs:    input.MemberIDs,
		WalletID:     input.WalletID,
		DueDay:       input.DueDay,
		Frequency:    input.Frequency,
		Recurrence:   input.Recurrence,
		StartMonth:   input.StartMonth,
		EndMonth:     input.EndMonth,
		Notes:        input.Notes,
	})
	if !ok {
		return
	}

	updated, err := h.Store.UpdateBill(r.Context(), id, bill)
	if err != nil {
		writeStoreError(w, err, "bill not found")
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// Delete handles DELETE /bills/{id}.
func (h *Bills) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}
	if err := h.Store.DeleteBill(r.Context(), id); err != nil {
		writeStoreError(w, err, "bill not found")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Bills) buildFromInput(w http.ResponseWriter, r *http.Request, input models.CreateBillInput) (models.Bill, bool) {
	if strings.TrimSpace(input.Name) == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return models.Bill{}, false
	}
	if input.DueDay < 1 || input.DueDay > 31 {
		http.Error(w, "due_day must be between 1 and 31", http.StatusBadRequest)
		return models.Bill{}, false
	}

	amountMode := models.NormalizeAmountMode(strings.ToLower(strings.TrimSpace(input.AmountMode)))
	interestRate := input.InterestRate

	amount := input.Amount
	if amountMode == models.BillAmountModeInterest {
		if amount <= 0 {
			http.Error(w, "amount must be greater than zero", http.StatusBadRequest)
			return models.Bill{}, false
		}
		if interestRate < 0 {
			http.Error(w, "interest_rate cannot be negative", http.StatusBadRequest)
			return models.Bill{}, false
		}
	} else if amount <= 0 {
		http.Error(w, "amount must be greater than zero", http.StatusBadRequest)
		return models.Bill{}, false
	}

	frequency := models.NormalizeFrequency(strings.ToLower(strings.TrimSpace(input.Frequency)))
	if !models.ValidBillFrequency(frequency) {
		http.Error(w, "frequency must be daily, weekdays, weekly, biweekly, monthly, or yearly", http.StatusBadRequest)
		return models.Bill{}, false
	}

	recurrence := strings.ToLower(strings.TrimSpace(input.Recurrence))
	if recurrence != models.BillRecurrenceOngoing && recurrence != models.BillRecurrenceUntil {
		http.Error(w, "recurrence must be ongoing or until", http.StatusBadRequest)
		return models.Bill{}, false
	}

	if _, err := time.Parse("2006-01", input.StartMonth); err != nil {
		http.Error(w, "start_month must be YYYY-MM", http.StatusBadRequest)
		return models.Bill{}, false
	}

	var normalizedEnd *string
	switch recurrence {
	case models.BillRecurrenceOngoing:
		normalizedEnd = nil
	case models.BillRecurrenceUntil:
		if input.EndMonth == nil || strings.TrimSpace(*input.EndMonth) == "" {
			http.Error(w, "end_month is required for until bills", http.StatusBadRequest)
			return models.Bill{}, false
		}
		trimmed := strings.TrimSpace(*input.EndMonth)
		if _, err := time.Parse("2006-01", trimmed); err != nil {
			http.Error(w, "end_month must be YYYY-MM", http.StatusBadRequest)
			return models.Bill{}, false
		}
		if trimmed < input.StartMonth {
			http.Error(w, "end_month must be on or after start_month", http.StatusBadRequest)
			return models.Bill{}, false
		}
		normalizedEnd = &trimmed
	}

	category, err := h.Categories.GetCategoryByID(r.Context(), input.CategoryID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			http.Error(w, "category not found", http.StatusBadRequest)
			return models.Bill{}, false
		}
		writeStoreError(w, err, "category not found")
		return models.Bill{}, false
	}

	memberIDs := uniquePositiveIDs(input.MemberIDs)
	for _, memberID := range memberIDs {
		if _, err := h.Members.GetMemberByID(r.Context(), memberID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, "member not found", http.StatusBadRequest)
				return models.Bill{}, false
			}
			writeStoreError(w, err, "member not found")
			return models.Bill{}, false
		}
	}

	walletID := input.WalletID
	if walletID == nil && isCardCategory(category.Name) && h.Wallets != nil {
		wallets, err := h.Wallets.ListWallets(r.Context())
		if err != nil {
			writeStoreError(w, err, "wallet not found")
			return models.Bill{}, false
		}
		for _, wallet := range wallets {
			if !models.IsCredit(wallet.Kind) {
				continue
			}
			if len(memberIDs) == 1 && wallet.MemberID != nil && *wallet.MemberID != memberIDs[0] {
				continue
			}
			id := wallet.ID
			walletID = &id
			break
		}
	}

	return models.Bill{
		Name:         strings.TrimSpace(input.Name),
		Amount:       amount,
		AmountMode:   amountMode,
		InterestRate: interestRate,
		CategoryID:   input.CategoryID,
		MemberIDs:    memberIDs,
		WalletID:     walletID,
		DueDay:       input.DueDay,
		Frequency:    frequency,
		Recurrence:   recurrence,
		StartMonth:   input.StartMonth,
		EndMonth:     normalizedEnd,
		Notes:        strings.TrimSpace(input.Notes),
	}, true
}

func isCardCategory(name string) bool {
	normalized := strings.NewReplacer("ã", "a", "á", "a", "â", "a").Replace(strings.ToLower(strings.TrimSpace(name)))
	return normalized == "cartao"
}

func uniquePositiveIDs(ids []int) []int {
	if len(ids) == 0 {
		return []int{}
	}
	seen := make(map[int]struct{}, len(ids))
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id < 1 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
