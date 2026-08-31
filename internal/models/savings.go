package models

import (
	"fmt"
	"math"
	"time"
)

const (
	SavingsEndNone   = "none"
	SavingsEndDate   = "date"
	SavingsEndAmount = "amount"

	CDIYieldFactor     = 1.03
	DefaultPlanMonths  = 12
	FallbackCDIAnnual  = 14.15 // % a.a. if BCB is unavailable
	businessDaysInYear = 252
)

// SavingsGoal is a joint family target for setting money aside.
type SavingsGoal struct {
	ID            int       `json:"id"`
	Name          string    `json:"name"`
	TargetAmount  float64   `json:"target_amount"`
	MonthlyAmount float64   `json:"monthly_amount"`
	SavedAmount   float64   `json:"saved_amount"`
	MemberIDs     []int     `json:"member_ids"`
	Notes         string    `json:"notes,omitempty"`
	EndKind       string    `json:"end_kind"`
	EndMonth      *string   `json:"end_month,omitempty"`
	CDIAnnual     float64   `json:"cdi_annual"`
	YieldAnnual   float64   `json:"yield_annual"`
	OpeningAmount float64   `json:"opening_amount"`
	CreatedAt     time.Time `json:"created_at"`
}

// CreateSavingsGoalInput is the JSON body for POST /savings.
type CreateSavingsGoalInput struct {
	Name         string  `json:"name"`
	TargetAmount float64 `json:"target_amount"`
	MemberIDs    []int   `json:"member_ids"`
	Notes        string  `json:"notes"`
	EndKind      string  `json:"end_kind"`
	EndMonth     *string `json:"end_month"`
}

// UpdateSavingsGoalInput is the JSON body for PUT /savings/{id}.
type UpdateSavingsGoalInput struct {
	Name         string  `json:"name"`
	TargetAmount float64 `json:"target_amount"`
	MemberIDs    []int   `json:"member_ids"`
	Notes        string  `json:"notes"`
	EndKind      string  `json:"end_kind"`
	EndMonth     *string `json:"end_month"`
}

// SavingsMonthAmount is how much was set aside for a goal in a month.
type SavingsMonthAmount struct {
	GoalID  int       `json:"goal_id"`
	Year    int       `json:"year"`
	Month   int       `json:"month"`
	Amount  float64   `json:"amount"`
	SavedAt time.Time `json:"saved_at"`
}

// SetSavingsMonthInput is the JSON body for PUT /savings/{id}/month.
type SetSavingsMonthInput struct {
	Year   int     `json:"year"`
	Month  int     `json:"month"`
	Amount float64 `json:"amount"`
}

// AdjustSavingsInput adds or removes money from a savings box.
type AdjustSavingsInput struct {
	Amount   float64 `json:"amount"`
	WalletID *int    `json:"wallet_id"`
}

// SavingsPlan is the monthly contribution computed at 103% of CDI.
type SavingsPlan struct {
	Months          int     `json:"months"`
	MonthlyAmount   float64 `json:"monthly_amount"`
	CDIAnnual       float64 `json:"cdi_annual"`
	YieldFactor     float64 `json:"yield_factor"`
	YieldAnnual     float64 `json:"yield_annual"`
	TargetAmount    float64 `json:"target_amount"`
	PerMember       float64 `json:"per_member"`
	MemberCount     int     `json:"member_count"`
	UsedDefaultTerm bool    `json:"used_default_term"`
}

// NormalizeEndKind returns a known end kind, defaulting empty to amount.
func NormalizeEndKind(value string) string {
	switch value {
	case SavingsEndNone, SavingsEndDate, SavingsEndAmount:
		return value
	case "":
		return SavingsEndAmount
	default:
		return ""
	}
}

// CDIDailyToAnnual converts a BCB CDI quote into % a.a.
// Values below 1 are treated as % per business day; otherwise as % a.a.
func CDIDailyToAnnual(value float64) float64 {
	if value <= 0 {
		return 0
	}
	if value < 1 {
		return (math.Pow(1+value/100, businessDaysInYear) - 1) * 100
	}
	return value
}

// MonthlyRateFromCDIAnnual is the monthly equivalent of 103% of CDI.
func MonthlyRateFromCDIAnnual(cdiAnnualPct float64) float64 {
	annual := (cdiAnnualPct / 100) * CDIYieldFactor
	if annual <= 0 {
		return 0
	}
	return math.Pow(1+annual, 1.0/12.0) - 1
}

// MonthsInclusive counts months from fromYear/fromMonth through endMonth (YYYY-MM).
func MonthsInclusive(fromYear, fromMonth int, endMonth string) (int, error) {
	end, err := time.Parse("2006-01", endMonth)
	if err != nil {
		return 0, fmt.Errorf("end_month must be YYYY-MM")
	}
	from := time.Date(fromYear, time.Month(fromMonth), 1, 0, 0, 0, 0, time.UTC)
	if end.Before(from) {
		return 0, fmt.Errorf("end_month must be on or after the current month")
	}
	months := (end.Year()-from.Year())*12 + int(end.Month()) - int(from.Month()) + 1
	if months < 1 {
		return 0, fmt.Errorf("end_month must be on or after the current month")
	}
	return months, nil
}

// MonthlyContribution is the PMT to reach remaining over n months at monthlyRate.
func MonthlyContribution(remaining float64, months int, monthlyRate float64) float64 {
	if remaining <= 0 || months < 1 {
		return 0
	}
	if monthlyRate <= 0 {
		return round2(remaining / float64(months))
	}
	factor := (math.Pow(1+monthlyRate, float64(months)) - 1) / monthlyRate
	if factor <= 0 {
		return round2(remaining / float64(months))
	}
	return round2(remaining / factor)
}

// BuildSavingsPlan computes the joint monthly deposit at 103% of CDI.
func BuildSavingsPlan(target, saved, cdiAnnual float64, months, memberCount int, usedDefault bool) SavingsPlan {
	if memberCount < 1 {
		memberCount = 1
	}
	remaining := target - saved
	if remaining < 0 {
		remaining = 0
	}
	monthly := MonthlyContribution(remaining, months, MonthlyRateFromCDIAnnual(cdiAnnual))
	yieldAnnual := round2(cdiAnnual * CDIYieldFactor)
	perMember := round2(monthly / float64(memberCount))
	return SavingsPlan{
		Months:          months,
		MonthlyAmount:   monthly,
		CDIAnnual:       round2(cdiAnnual),
		YieldFactor:     CDIYieldFactor,
		YieldAnnual:     yieldAnnual,
		TargetAmount:    target,
		PerMember:       perMember,
		MemberCount:     memberCount,
		UsedDefaultTerm: usedDefault,
	}
}

func (g *SavingsGoal) ApplyYield() {
	g.YieldAnnual = round2(g.CDIAnnual * CDIYieldFactor)
	if g.EndKind == "" {
		g.EndKind = SavingsEndAmount
	}
}

// MonthlyPlan is the amount reserved in the forecast for year/month.
func (g SavingsGoal) MonthlyPlan(year, month int) float64 {
	if g.EndKind == SavingsEndNone {
		return 0
	}
	if g.TargetAmount > 0 && g.SavedAmount >= g.TargetAmount {
		return 0
	}
	if g.EndKind == SavingsEndDate && g.EndMonth != nil && *g.EndMonth != "" {
		current := fmt.Sprintf("%04d-%02d", year, month)
		if current > *g.EndMonth {
			return 0
		}
	}
	remaining := g.TargetAmount - g.SavedAmount
	if remaining <= 0 {
		return 0
	}
	if g.MonthlyAmount <= 0 {
		return 0
	}
	if g.MonthlyAmount < remaining {
		return g.MonthlyAmount
	}
	return remaining
}

// ProjectedBalance is the box value after months of deposits at 103% of CDI.
func ProjectedBalance(saved, monthly, cdiAnnual float64, months int) float64 {
	if months < 1 {
		return round2(saved)
	}
	rate := MonthlyRateFromCDIAnnual(cdiAnnual)
	value := saved
	for i := 0; i < months; i++ {
		value = value*(1+rate) + monthly
	}
	return round2(value)
}
