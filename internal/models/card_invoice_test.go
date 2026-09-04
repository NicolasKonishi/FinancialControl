package models_test

import (
	"testing"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

func TestCardCycleClosing14Due21(t *testing.T) {
	closing, due := 14, 21
	wallet := models.Wallet{ClosingDay: &closing, DueDay: &due}
	tests := []struct {
		date string
		want string
	}{
		{date: "2026-09-13", want: "2026-09-21"},
		{date: "2026-09-14", want: "2026-10-21"},
		{date: "2026-09-30", want: "2026-10-21"},
	}
	for _, tc := range tests {
		purchase, _ := time.Parse("2006-01-02", tc.date)
		cycle := models.CardCycleForPurchase(wallet, purchase)
		if got := cycle.DueDate.Format("2006-01-02"); got != tc.want {
			t.Errorf("purchase %s due = %s, want %s", tc.date, got, tc.want)
		}
	}
}

func TestApplyPlannedCardBillsRebuildsCalculatedInvoice(t *testing.T) {
	walletID := 10
	end := "2026-12"
	bills := []models.Bill{
		{
			Name: "Mercadolivre", Amount: 381.26, WalletID: &walletID, DueDay: 21,
			Frequency: models.BillFrequencyMonthly, Recurrence: models.BillRecurrenceUntil,
			StartMonth: "2026-10", EndMonth: &end, Source: models.BillSourceStatement,
		},
		{
			Name: "NU Plus", Amount: 29, WalletID: &walletID, DueDay: 21,
			Frequency: models.BillFrequencyMonthly, Recurrence: models.BillRecurrenceOngoing,
			StartMonth: "2026-09", Source: models.BillSourceManual,
		},
	}
	calculated := models.ApplyPlannedCardBills(models.CardInvoice{
		WalletID: walletID, Year: 2026, Month: 11, Amount: 381.26, Source: "calculated",
	}, bills)
	if calculated.Amount != 410.26 {
		t.Fatalf("november forecast = %v, want 410.26", calculated.Amount)
	}

	legacy := models.ApplyPlannedCardBills(models.CardInvoice{
		WalletID: walletID, Year: 2026, Month: 11, Amount: 200, Source: "calculated",
	}, nil)
	if legacy.Amount != 200 {
		t.Fatalf("calculated invoice without bills should keep stored amount, got %v", legacy.Amount)
	}

	statement := models.ApplyPlannedCardBills(models.CardInvoice{
		WalletID: walletID, Year: 2026, Month: 10, Amount: 456.26, Source: "statement",
	}, bills)
	if statement.Amount != 485.26 {
		t.Fatalf("october statement should add manual NU Plus: got %v, want 485.26", statement.Amount)
	}
}

func TestCardCycleDueBeforeClosing(t *testing.T) {
	closing, due := 25, 5
	wallet := models.Wallet{ClosingDay: &closing, DueDay: &due}
	purchase, _ := time.Parse("2006-01-02", "2026-09-10")
	cycle := models.CardCycleForPurchase(wallet, purchase)
	if got := cycle.DueDate.Format("2006-01-02"); got != "2026-10-05" {
		t.Fatalf("due = %s, want 2026-10-05", got)
	}
}
