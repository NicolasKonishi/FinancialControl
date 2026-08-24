package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// AnalysisClient is the dependency used to call the Python analysis service.
type AnalysisClient interface {
	MonthlyAnalysis(ctx context.Context, req models.MonthlyAnalysisRequest) (models.MonthlyAnalysisResponse, error)
}

// MonthlyTransactionStore loads transactions for a specific month.
type MonthlyTransactionStore interface {
	ListTransactionsByMonth(ctx context.Context, year, month int) ([]models.Transaction, error)
}

// Analysis handles analysis endpoints that orchestrate Go + Python.
type Analysis struct {
	Transactions MonthlyTransactionStore
	Categories   CategoryStore
	Client       AnalysisClient
}

// Monthly handles GET /analysis/monthly.
func (h *Analysis) Monthly(w http.ResponseWriter, r *http.Request) {
	year, month, ok := parseYearMonth(w, r)
	if !ok {
		return
	}

	transactions, err := h.Transactions.ListTransactionsByMonth(r.Context(), year, month)
	if err != nil {
		writeStoreError(w, err, "transaction not found")
		return
	}

	categories, err := h.Categories.ListCategories(r.Context())
	if err != nil {
		writeStoreError(w, err, "category not found")
		return
	}

	result, err := h.Client.MonthlyAnalysis(r.Context(), models.MonthlyAnalysisRequest{
		Year:         year,
		Month:        month,
		Transactions: transactions,
		Categories:   categories,
	})
	if err != nil {
		http.Error(w, "analysis service unavailable: "+err.Error(), http.StatusBadGateway)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

func parseYearMonth(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	now := time.Now().UTC()
	year := now.Year()
	month := int(now.Month())

	if y := r.URL.Query().Get("year"); y != "" {
		parsed, err := strconv.Atoi(y)
		if err != nil || parsed < 1 {
			http.Error(w, "invalid year", http.StatusBadRequest)
			return 0, 0, false
		}
		year = parsed
	}
	if m := r.URL.Query().Get("month"); m != "" {
		parsed, err := strconv.Atoi(m)
		if err != nil || parsed < 1 || parsed > 12 {
			http.Error(w, "invalid month", http.StatusBadRequest)
			return 0, 0, false
		}
		month = parsed
	}
	return year, month, true
}
