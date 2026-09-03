package statement_test

import (
	"testing"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
	"github.com/NicolasKonishi/FinancialControl/internal/statement"
)

func TestBuildPreviewMarksExistingPurchases(t *testing.T) {
	walletID := 3
	parsed := models.ParsedStatement{
		Issuer: "nubank",
		Items: []models.ParsedStatementItem{
			{Date: "2026-07-10", Description: "IFOOD *IFOOD", Amount: 42.9, Kind: "expense", SuggestedIcon: "food"},
			{Date: "2026-07-11", Description: "UBER TRIP", Amount: 18.5, Kind: "expense", SuggestedIcon: "transport"},
			{Date: "2026-07-08", Description: "Pagamento recebido", Amount: 1234.56, Kind: "payment", SuggestedIcon: "other"},
		},
	}
	existing := []models.Transaction{
		{
			ID:          11,
			Description: "iFood",
			Amount:      42.90,
			Date:        time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC),
			WalletID:    &walletID,
		},
	}
	cats := []models.Category{
		{ID: 1, Name: "Comida", Icon: "food"},
		{ID: 2, Name: "Transporte", Icon: "transport"},
		{ID: 3, Name: "Outro", Icon: "other"},
	}

	preview := statement.BuildPreview(parsed, existing, cats, &walletID, nil)
	if preview.NewCount != 1 {
		t.Fatalf("new_count = %d, want 1", preview.NewCount)
	}
	if preview.MatchedCount != 1 {
		t.Fatalf("matched_count = %d, want 1", preview.MatchedCount)
	}
	if preview.SkippedCount != 1 {
		t.Fatalf("skipped_count = %d, want 1", preview.SkippedCount)
	}
	if !preview.Items[0].AlreadyRecorded || preview.Items[0].Selected {
		t.Fatalf("ifood should be matched and unselected: %+v", preview.Items[0])
	}
	if !preview.Items[1].Selected || preview.Items[1].CategoryID != 2 {
		t.Fatalf("uber should be selected in transport: %+v", preview.Items[1])
	}
	if preview.Items[2].Selected || preview.Items[2].Kind != "payment" {
		t.Fatalf("payment should stay skipped: %+v", preview.Items[2])
	}
}

func TestBuildPreviewSelectsIncomeAndSkipsTransfer(t *testing.T) {
	parsed := models.ParsedStatement{
		Issuer: "nubank",
		Items: []models.ParsedStatementItem{
			{Date: "2026-09-01", Description: "EMPRESA EXEMPLO LTDA", Amount: 2000, Kind: "income", SuggestedIcon: "salary"},
			{Date: "2026-09-01", Description: "Aplicação RDB", Amount: 2000, Kind: "transfer", SuggestedIcon: "other"},
			{Date: "2026-09-01", Description: "SAO ROQUE", Amount: 43.46, Kind: "expense", SuggestedIcon: "other"},
		},
	}
	cats := []models.Category{
		{ID: 8, Name: "Salário", Icon: "salary"},
		{ID: 3, Name: "Outro", Icon: "other"},
	}
	preview := statement.BuildPreview(parsed, nil, cats, nil, nil)
	if preview.NewCount != 2 {
		t.Fatalf("new_count = %d, want 2", preview.NewCount)
	}
	if preview.SkippedCount != 1 {
		t.Fatalf("skipped_count = %d, want 1", preview.SkippedCount)
	}
	if !preview.Items[0].Selected || preview.Items[0].CategoryID != 8 {
		t.Fatalf("income should be selected as salary: %+v", preview.Items[0])
	}
	if preview.Items[1].Selected || preview.Items[1].Kind != "transfer" {
		t.Fatalf("RDB application should stay skipped: %+v", preview.Items[1])
	}
	if !preview.Items[2].Selected {
		t.Fatalf("debit purchase should be selected: %+v", preview.Items[2])
	}
}

func TestCategoryForIconFallsBack(t *testing.T) {
	cats := []models.Category{
		{ID: 8, Name: "Salário", Icon: "salary"},
		{ID: 2, Name: "Mercado", Icon: "market"},
	}
	if got := statement.CategoryForIcon(cats, "pets"); got != 2 {
		t.Fatalf("fallback = %d, want 2", got)
	}
}
