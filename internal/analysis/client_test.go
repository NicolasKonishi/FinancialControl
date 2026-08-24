package analysis_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/NicolasKonishi/FinancialControl/internal/analysis"
	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

func TestClientMonthlyAnalysis(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/analysis/monthly" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(models.MonthlyAnalysisResponse{
			Year:         2026,
			Month:        8,
			TotalIncome:  100,
			TotalExpense: 40,
			Balance:      60,
			ByCategory: []models.CategoryBreakdown{
				{CategoryID: 1, CategoryName: "Food", Total: 40},
			},
			TransactionCount: 2,
		})
	}))
	defer server.Close()

	client := analysis.NewClient(server.URL)
	result, err := client.MonthlyAnalysis(context.Background(), models.MonthlyAnalysisRequest{
		Year:  2026,
		Month: 8,
	})
	if err != nil {
		t.Fatalf("MonthlyAnalysis error: %v", err)
	}
	if result.Balance != 60 {
		t.Errorf("balance = %v, want 60", result.Balance)
	}
}
