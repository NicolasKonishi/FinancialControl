package models

import (
	"fmt"
	"math"
	"time"
)

// Bill amount modes.
const (
	BillAmountModeFixed    = "fixed"
	BillAmountModeInterest = "interest" // monthly % on top of base amount (evolução de obra)
)

// NormalizeAmountMode returns a known mode, defaulting empty to fixed.
func NormalizeAmountMode(value string) string {
	switch value {
	case BillAmountModeInterest, "schedule": // legacy schedule → interest
		return BillAmountModeInterest
	default:
		return BillAmountModeFixed
	}
}

func monthKey(year, month int) string {
	return fmt.Sprintf("%04d-%02d", year, month)
}

func monthsSinceStart(startMonth string, year, month int) (int, bool) {
	start, err := time.Parse("2006-01", startMonth)
	if err != nil {
		return 0, false
	}
	target := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	if target.Before(start) {
		return 0, false
	}
	return (target.Year()-start.Year())*12 + int(target.Month()-start.Month()), true
}

func (b Bill) interestChargeForMonth(year, month int) float64 {
	n, ok := monthsSinceStart(b.StartMonth, year, month)
	if !ok || b.Amount <= 0 {
		return 0
	}
	if b.InterestRate <= 0 {
		return b.Amount
	}
	factor := math.Pow(1+b.InterestRate/100, float64(n))
	return round2(b.Amount * factor)
}
