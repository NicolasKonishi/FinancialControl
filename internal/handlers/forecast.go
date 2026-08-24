package handlers

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// ForecastStore provides data needed to compute a monthly forecast.
type ForecastStore interface {
	SumMonthlySalaries(ctx context.Context) (float64, error)
	ListTransactionsByMonth(ctx context.Context, year, month int) ([]models.Transaction, error)
}

// Forecast handles budget forecast endpoints.
type Forecast struct {
	Store ForecastStore
}

// Monthly handles GET /forecast/monthly.
func (h *Forecast) Monthly(w http.ResponseWriter, r *http.Request) {
	year, month, ok := parseYearMonth(w, r)
	if !ok {
		return
	}

	plannedSalary, err := h.Store.SumMonthlySalaries(r.Context())
	if err != nil {
		writeStoreError(w, err, "not found")
		return
	}

	transactions, err := h.Store.ListTransactionsByMonth(r.Context(), year, month)
	if err != nil {
		writeStoreError(w, err, "not found")
		return
	}

	var extraIncome, totalExpense float64
	for _, tx := range transactions {
		switch tx.Type {
		case models.TransactionTypeIncome:
			extraIncome += tx.Amount
		case models.TransactionTypeExpense:
			totalExpense += tx.Amount
		}
	}

	now := time.Now().UTC()
	daysInMonth := daysIn(year, month)
	daysElapsed := daysInMonth
	if now.Year() == year && int(now.Month()) == month {
		daysElapsed = now.Day()
	} else if time.Date(year, time.Month(month), daysInMonth, 23, 59, 59, 0, time.UTC).Before(now) {
		daysElapsed = daysInMonth
	} else {
		daysElapsed = 0
	}
	if daysElapsed < 1 && (now.Year() == year && int(now.Month()) == month) {
		daysElapsed = 1
	}
	daysRemaining := daysInMonth - daysElapsed
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	totalAvailable := plannedSalary + extraIncome
	remaining := totalAvailable - totalExpense

	projectedExpense := totalExpense
	if daysElapsed > 0 {
		projectedExpense = (totalExpense / float64(daysElapsed)) * float64(daysInMonth)
	}

	safeDaily := 0.0
	if daysRemaining > 0 {
		safeDaily = remaining / float64(daysRemaining)
	}
	if safeDaily < 0 {
		safeDaily = 0
	}

	pace := 0.0
	if totalAvailable > 0 {
		pace = projectedExpense / totalAvailable
	}

	writeJSON(w, http.StatusOK, models.MonthlyForecast{
		Year:             year,
		Month:            month,
		PlannedSalary:    round2(plannedSalary),
		ExtraIncome:      round2(extraIncome),
		TotalAvailable:   round2(totalAvailable),
		TotalExpense:     round2(totalExpense),
		Remaining:        round2(remaining),
		DaysInMonth:      daysInMonth,
		DaysElapsed:      daysElapsed,
		DaysRemaining:    daysRemaining,
		ProjectedExpense: round2(projectedExpense),
		SafeDailySpend:   round2(safeDaily),
		ExpensePaceRatio: round2(pace),
	})
}

func daysIn(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
