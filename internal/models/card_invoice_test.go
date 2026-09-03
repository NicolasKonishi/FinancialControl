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

func TestCardCycleDueBeforeClosing(t *testing.T) {
	closing, due := 25, 5
	wallet := models.Wallet{ClosingDay: &closing, DueDay: &due}
	purchase, _ := time.Parse("2006-01-02", "2026-09-10")
	cycle := models.CardCycleForPurchase(wallet, purchase)
	if got := cycle.DueDate.Format("2006-01-02"); got != "2026-10-05" {
		t.Fatalf("due = %s, want 2026-10-05", got)
	}
}
