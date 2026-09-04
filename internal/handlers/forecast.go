package handlers

import (
	"context"
	"math"
	"net/http"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
	"github.com/NicolasKonishi/FinancialControl/internal/statement"
)

// ForecastStore provides data needed to compute a monthly forecast.
type ForecastStore interface {
	ListMembers(ctx context.Context) ([]models.Member, error)
	ListTransactionsByMonth(ctx context.Context, year, month int) ([]models.Transaction, error)
	ListBillsActiveInMonth(ctx context.Context, year, month int) ([]models.Bill, error)
	ListSavingsGoals(ctx context.Context) ([]models.SavingsGoal, error)
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

	members, err := h.Store.ListMembers(r.Context())
	if err != nil {
		writeStoreError(w, err, "not found")
		return
	}

	transactions, err := h.Store.ListTransactionsByMonth(r.Context(), year, month)
	if err != nil {
		writeStoreError(w, err, "not found")
		return
	}

	bills, err := h.Store.ListBillsActiveInMonth(r.Context(), year, month)
	if err != nil {
		writeStoreError(w, err, "not found")
		return
	}

	goals, err := h.Store.ListSavingsGoals(r.Context())
	if err != nil {
		writeStoreError(w, err, "not found")
		return
	}

	var plannedSalary, extraIncome, variableExpense, plannedBills, plannedSavings float64
	for _, member := range members {
		plannedSalary += member.MonthlySalary
	}
	coveredTx := map[int]bool{}
	usedBills := map[int]bool{}
	for _, tx := range transactions {
		switch tx.Type {
		case models.TransactionTypeIncome:
			extraIncome += tx.Amount
		case models.TransactionTypeExpense:
			if statement.ExpenseCoveredByBill(tx, bills, year, month, usedBills) {
				coveredTx[tx.ID] = true
				continue
			}
			variableExpense += tx.Amount
		}
	}
	for _, bill := range bills {
		plannedBills += bill.ChargeForMonth(year, month)
	}
	for _, goal := range goals {
		plannedSavings += goal.MonthlyPlan(year, month)
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
	totalExpense := variableExpense + plannedBills
	remaining := totalAvailable - totalExpense - plannedSavings

	projectedVariable := variableExpense
	if daysElapsed > 0 {
		projectedVariable = (variableExpense / float64(daysElapsed)) * float64(daysInMonth)
	}
	projectedExpense := projectedVariable + plannedBills

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
		PlannedBills:     round2(plannedBills),
		PlannedSavings:   round2(plannedSavings),
		TotalExpense:     round2(totalExpense),
		Remaining:        round2(remaining),
		DaysInMonth:      daysInMonth,
		DaysElapsed:      daysElapsed,
		DaysRemaining:    daysRemaining,
		ProjectedExpense: round2(projectedExpense),
		SafeDailySpend:   round2(safeDaily),
		ExpensePaceRatio: round2(pace),
		ByMember:         buildMemberForecasts(members, transactions, bills, goals, year, month, coveredTx),
	})
}

func buildMemberForecasts(
	members []models.Member,
	transactions []models.Transaction,
	bills []models.Bill,
	goals []models.SavingsGoal,
	year, month int,
	coveredTx map[int]bool,
) []models.MemberForecast {
	out := make([]models.MemberForecast, 0, len(members))
	for _, member := range members {
		var extraIncome, variableExpense, billShare, savingsShare float64
		for _, tx := range transactions {
			if tx.MemberID == nil || *tx.MemberID != member.ID {
				continue
			}
			switch tx.Type {
			case models.TransactionTypeIncome:
				extraIncome += tx.Amount
			case models.TransactionTypeExpense:
				if coveredTx[tx.ID] {
					continue
				}
				variableExpense += tx.Amount
			}
		}
		for _, bill := range bills {
			share := billShareForMember(bill, member.ID, year, month)
			billShare += share
		}
		for _, goal := range goals {
			savingsShare += savingsShareForMember(goal, member.ID, year, month)
		}

		available := member.MonthlySalary + extraIncome
		toPay := billShare + variableExpense
		out = append(out, models.MemberForecast{
			MemberID:        member.ID,
			MemberName:      member.Name,
			PlannedSalary:   round2(member.MonthlySalary),
			ExtraIncome:     round2(extraIncome),
			TotalAvailable:  round2(available),
			BillShare:       round2(billShare),
			SavingsShare:    round2(savingsShare),
			VariableExpense: round2(variableExpense),
			TotalToPay:      round2(toPay),
			Remaining:       round2(available - toPay - savingsShare),
		})
	}
	return out
}

// billShareForMember splits the month charge equally across payers.
func billShareForMember(bill models.Bill, memberID, year, month int) float64 {
	if len(bill.MemberIDs) == 0 {
		return 0
	}
	found := false
	for _, id := range bill.MemberIDs {
		if id == memberID {
			found = true
			break
		}
	}
	if !found {
		return 0
	}
	return bill.ChargeForMonth(year, month) / float64(len(bill.MemberIDs))
}

func savingsShareForMember(goal models.SavingsGoal, memberID, year, month int) float64 {
	if len(goal.MemberIDs) == 0 {
		return 0
	}
	found := false
	for _, id := range goal.MemberIDs {
		if id == memberID {
			found = true
			break
		}
	}
	if !found {
		return 0
	}
	return goal.MonthlyPlan(year, month) / float64(len(goal.MemberIDs))
}

func daysIn(year, month int) int {
	return time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
