package handlers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
	"github.com/NicolasKonishi/FinancialControl/internal/repository"
)

// ListPayments handles GET /bills/payments?year=&month=.
func (h *Bills) ListPayments(w http.ResponseWriter, r *http.Request) {
	year, month, ok := parseYearMonth(w, r)
	if !ok {
		return
	}

	items, err := h.Store.ListBillPayments(r.Context(), year, month)
	if err != nil {
		writeStoreError(w, err, "bill not found")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// SetPaid handles PUT /bills/{id}/paid.
func (h *Bills) SetPaid(w http.ResponseWriter, r *http.Request) {
	id, ok := parsePositiveID(w, r.PathValue("id"))
	if !ok {
		return
	}

	var input models.SetBillPaidInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}
	if input.Year < 1 {
		http.Error(w, "invalid year", http.StatusBadRequest)
		return
	}
	if input.Month < 1 || input.Month > 12 {
		http.Error(w, "invalid month", http.StatusBadRequest)
		return
	}

	if input.Paid {
		if input.PaidByMemberID == nil || *input.PaidByMemberID < 1 {
			http.Error(w, "paid_by_member_id is required", http.StatusBadRequest)
			return
		}
		if _, err := h.Members.GetMemberByID(r.Context(), *input.PaidByMemberID); err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				http.Error(w, "member not found", http.StatusBadRequest)
				return
			}
			writeStoreError(w, err, "member not found")
			return
		}
	}

	if err := h.Store.SetBillPaid(r.Context(), id, input.Year, input.Month, input.Paid, input.PaidByMemberID, input.WalletID); err != nil {
		if errors.Is(err, repository.ErrNoWallet) {
			http.Error(w, "this person has no account to debit", http.StatusBadRequest)
			return
		}
		if errors.Is(err, repository.ErrWalletOwner) {
			http.Error(w, "wallet does not belong to this person", http.StatusBadRequest)
			return
		}
		writeStoreError(w, err, "bill not found")
		return
	}

	if !input.Paid {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	items, err := h.Store.ListBillPayments(r.Context(), input.Year, input.Month)
	if err != nil {
		writeStoreError(w, err, "bill not found")
		return
	}
	for _, item := range items {
		if item.BillID == id {
			writeJSON(w, http.StatusOK, item)
			return
		}
	}
	writeJSON(w, http.StatusOK, models.BillPayment{
		BillID: id,
		Year:   input.Year,
		Month:  input.Month,
	})
}
