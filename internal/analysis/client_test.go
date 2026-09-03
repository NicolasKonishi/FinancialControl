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

func TestClientParseStatement(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/statements/parse" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if r.URL.Query().Get("year") != "2026" || r.URL.Query().Get("month") != "8" {
			t.Fatalf("unexpected query %s", r.URL.RawQuery)
		}
		_ = json.NewEncoder(w).Encode(models.ParsedStatement{
			Issuer: "nubank",
			Items: []models.ParsedStatementItem{
				{Date: "2026-07-10", Description: "IFOOD", Amount: 42.9, Kind: "expense", SuggestedIcon: "food"},
			},
		})
	}))
	defer server.Close()

	client := analysis.NewClient(server.URL)
	result, err := client.ParseStatement(context.Background(), []byte("%PDF"), 2026, 8)
	if err != nil {
		t.Fatalf("ParseStatement error: %v", err)
	}
	if result.Issuer != "nubank" || len(result.Items) != 1 {
		t.Fatalf("unexpected parse result: %+v", result)
	}
}
