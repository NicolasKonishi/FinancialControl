package models

// MonthlyForecast summarizes salary-based budget expectations for a month.
type MonthlyForecast struct {
	Year             int     `json:"year"`
	Month            int     `json:"month"`
	PlannedSalary    float64 `json:"planned_salary"`
	ExtraIncome      float64 `json:"extra_income"`
	TotalAvailable   float64 `json:"total_available"`
	TotalExpense     float64 `json:"total_expense"`
	Remaining        float64 `json:"remaining"`
	DaysInMonth      int     `json:"days_in_month"`
	DaysElapsed      int     `json:"days_elapsed"`
	DaysRemaining    int     `json:"days_remaining"`
	ProjectedExpense float64 `json:"projected_expense"`
	SafeDailySpend   float64 `json:"safe_daily_spend"`
	ExpensePaceRatio float64 `json:"expense_pace_ratio"`
}
