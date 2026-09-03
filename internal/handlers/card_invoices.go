package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// CardInvoiceStore reads invoices for a due month.
type CardInvoiceStore interface {
	ListCardInvoices(ctx context.Context, year, month int) ([]models.CardInvoice, error)
}

// CardInvoices handles monthly credit-card invoices.
type CardInvoices struct {
	Store CardInvoiceStore
}

// List handles GET /card-invoices?year=YYYY&month=M.
func (h *CardInvoices) List(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	year := queryInt(r, "year", now.Year())
	month := queryInt(r, "month", int(now.Month()))
	if year < 2000 || month < 1 || month > 12 {
		http.Error(w, "invalid year or month", http.StatusBadRequest)
		return
	}
	items, err := h.Store.ListCardInvoices(r.Context(), year, month)
	if err != nil {
		writeStoreError(w, err, "invoice not found")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func queryInt(r *http.Request, key string, fallback int) int {
	raw := r.URL.Query().Get(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}
