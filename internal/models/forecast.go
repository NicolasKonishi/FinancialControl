package models

// MonthlyForecast summarizes salary-based budget expectations for a month.
type MonthlyForecast struct {
	Year             int              `json:"year"`
	Month            int              `json:"month"`
	PlannedSalary    float64          `json:"planned_salary"`
	ExtraIncome      float64          `json:"extra_income"`
	TotalAvailable   float64          `json:"total_available"`
	PlannedBills     float64          `json:"planned_bills"`
	PlannedSavings   float64          `json:"planned_savings"`
	TotalExpense     float64          `json:"total_expense"`
	Remaining        float64          `json:"remaining"`
	DaysInMonth      int              `json:"days_in_month"`
	DaysElapsed      int              `json:"days_elapsed"`
	DaysRemaining    int              `json:"days_remaining"`
	ProjectedExpense float64          `json:"projected_expense"`
	SafeDailySpend   float64          `json:"safe_daily_spend"`
	ExpensePaceRatio float64          `json:"expense_pace_ratio"`
	ByMember         []MemberForecast `json:"by_member"`
}

// MemberForecast breaks down the month for one family member.
type MemberForecast struct {
	MemberID        int     `json:"member_id"`
	MemberName      string  `json:"member_name"`
	PlannedSalary   float64 `json:"planned_salary"`
	ExtraIncome     float64 `json:"extra_income"`
	TotalAvailable  float64 `json:"total_available"`
	BillShare       float64 `json:"bill_share"`
	SavingsShare    float64 `json:"savings_share"`
	VariableExpense float64 `json:"variable_expense"`
	TotalToPay      float64 `json:"total_to_pay"`
	Remaining       float64 `json:"remaining"`
}
