package models

import (
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Bill end-condition kinds (when the series stops).
const (
	BillRecurrenceOngoing = "ongoing" // no end date (internet, electricity…)
	BillRecurrenceUntil   = "until"   // ends on end_month (subscription, credit…)
)

// Bill frequency kinds (how often each charge happens).
const (
	BillFrequencyDaily    = "daily"
	BillFrequencyWeekdays = "weekdays" // Mon–Fri
	BillFrequencyWeekly   = "weekly"
	BillFrequencyBiweekly = "biweekly"
	BillFrequencyMonthly  = "monthly"
	BillFrequencyYearly   = "yearly"
)

const (
	BillSourceManual    = "manual"
	BillSourceStatement = "statement"
)

var installmentRe = regexp.MustCompile(`(?i)(?:parcela\s*)?(\d{1,3})\s*(?:/|de)\s*(\d{1,3})\b`)

// Amount is the base charge when fixed, or the 1st month value when amount_mode is interest.
type Bill struct {
	ID               int       `json:"id"`
	Name             string    `json:"name"`
	Amount           float64   `json:"amount"`
	AmountMode       string    `json:"amount_mode"`
	InterestRate     float64   `json:"interest_rate"`
	CategoryID       int       `json:"category_id"`
	MemberIDs        []int     `json:"member_ids"`
	WalletID         *int      `json:"wallet_id"`
	DueDay           int       `json:"due_day"`
	Frequency        string    `json:"frequency"`
	Recurrence       string    `json:"recurrence"`
	StartMonth       string    `json:"start_month"`
	EndMonth         *string   `json:"end_month,omitempty"`
	Notes            string    `json:"notes,omitempty"`
	Source           string    `json:"source"`
	InstallmentStart int       `json:"installment_start,omitempty"`
	InstallmentTotal int       `json:"installment_total,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// CreateBillInput is the JSON body for POST /bills.
type CreateBillInput struct {
	Name             string  `json:"name"`
	Amount           float64 `json:"amount"`
	AmountMode       string  `json:"amount_mode"`
	InterestRate     float64 `json:"interest_rate"`
	CategoryID       int     `json:"category_id"`
	MemberIDs        []int   `json:"member_ids"`
	WalletID         *int    `json:"wallet_id"`
	DueDay           int     `json:"due_day"`
	Frequency        string  `json:"frequency"`
	Recurrence       string  `json:"recurrence"`
	StartMonth       string  `json:"start_month"`
	EndMonth         *string `json:"end_month"`
	Notes            string  `json:"notes"`
	Source           string  `json:"source"`
	InstallmentStart int     `json:"installment_start"`
	InstallmentTotal int     `json:"installment_total"`
}

// UpdateBillInput is the JSON body for PUT /bills/{id}.
type UpdateBillInput struct {
	Name             string  `json:"name"`
	Amount           float64 `json:"amount"`
	AmountMode       string  `json:"amount_mode"`
	InterestRate     float64 `json:"interest_rate"`
	CategoryID       int     `json:"category_id"`
	MemberIDs        []int   `json:"member_ids"`
	WalletID         *int    `json:"wallet_id"`
	DueDay           int     `json:"due_day"`
	Frequency        string  `json:"frequency"`
	Recurrence       string  `json:"recurrence"`
	StartMonth       string  `json:"start_month"`
	EndMonth         *string `json:"end_month"`
	Notes            string  `json:"notes"`
	Source           string  `json:"source"`
	InstallmentStart int     `json:"installment_start"`
	InstallmentTotal int     `json:"installment_total"`
}

// ValidBillFrequency reports whether frequency is supported.
func ValidBillFrequency(value string) bool {
	switch value {
	case BillFrequencyDaily, BillFrequencyWeekdays, BillFrequencyWeekly,
		BillFrequencyBiweekly, BillFrequencyMonthly, BillFrequencyYearly:
		return true
	default:
		return false
	}
}

// NormalizeFrequency returns a known frequency, defaulting empty to monthly.
func NormalizeFrequency(value string) string {
	if value == "" {
		return BillFrequencyMonthly
	}
	return value
}

// ParseInstallment recognizes descriptions such as "Notebook 1/3" and returns
// the clean purchase name plus the current and total installment numbers.
func ParseInstallment(value string) (string, int, int, bool) {
	match := installmentRe.FindStringSubmatchIndex(value)
	if match == nil {
		return strings.TrimSpace(value), 0, 0, false
	}
	current, currentErr := strconv.Atoi(value[match[2]:match[3]])
	total, totalErr := strconv.Atoi(value[match[4]:match[5]])
	if currentErr != nil || totalErr != nil || current < 1 || total < current || total > 120 {
		return strings.TrimSpace(value), 0, 0, false
	}
	name := strings.TrimSpace(value[:match[0]] + " " + value[match[1]:])
	name = strings.Trim(name, " -–—·()[]")
	name = strings.Join(strings.Fields(name), " ")
	if name == "" {
		name = strings.TrimSpace(value)
	}
	return name, current, total, true
}

// AddMonthsToMonthKey shifts a YYYY-MM competence by delta months.
func AddMonthsToMonthKey(value string, delta int) (string, bool) {
	month, err := time.Parse("2006-01", value)
	if err != nil {
		return "", false
	}
	return month.AddDate(0, delta, 0).Format("2006-01"), true
}

// IsActiveInMonth reports whether the bill has at least one charge in year/month.
func (b Bill) IsActiveInMonth(year, month int) bool {
	return b.OccurrencesInMonth(year, month) > 0
}

// ChargeForMonth is the total charged in that month.
func (b Bill) ChargeForMonth(year, month int) float64 {
	if b.OccurrencesInMonth(year, month) == 0 {
		return 0
	}
	if NormalizeAmountMode(b.AmountMode) == BillAmountModeInterest {
		return b.interestChargeForMonth(year, month)
	}
	return b.Amount * float64(b.OccurrencesInMonth(year, month))
}

// OccurrencesInMonth counts how many times this bill charges in year/month.
func (b Bill) OccurrencesInMonth(year, month int) int {
	anchor, ok := b.anchorDate()
	if !ok {
		return 0
	}

	monthStart := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC)

	rangeStart := monthStart
	if anchor.After(rangeStart) {
		rangeStart = anchor
	}
	rangeEnd := monthEnd
	if end := b.endDate(); end != nil && end.Before(rangeEnd) {
		rangeEnd = *end
	}
	if rangeStart.After(rangeEnd) {
		return 0
	}

	switch NormalizeFrequency(b.Frequency) {
	case BillFrequencyDaily:
		return daysBetweenInclusive(rangeStart, rangeEnd)
	case BillFrequencyWeekdays:
		return countWeekdays(rangeStart, rangeEnd)
	case BillFrequencyWeekly:
		return countEveryNDays(anchor, rangeStart, rangeEnd, 7)
	case BillFrequencyBiweekly:
		return countEveryNDays(anchor, rangeStart, rangeEnd, 14)
	case BillFrequencyYearly:
		if time.Month(month) != anchor.Month() {
			return 0
		}
		due := clampDay(year, month, b.DueDay)
		if due.Before(rangeStart) || due.After(rangeEnd) {
			return 0
		}
		return 1
	default: // monthly
		due := clampDay(year, month, b.DueDay)
		if due.Before(rangeStart) || due.After(rangeEnd) {
			return 0
		}
		return 1
	}
}

func (b Bill) anchorDate() (time.Time, bool) {
	start, err := time.Parse("2006-01", b.StartMonth)
	if err != nil {
		return time.Time{}, false
	}
	return clampDay(start.Year(), int(start.Month()), b.DueDay), true
}

func (b Bill) endDate() *time.Time {
	if b.Recurrence == BillRecurrenceOngoing || b.EndMonth == nil || *b.EndMonth == "" {
		return nil
	}
	end, err := time.Parse("2006-01", *b.EndMonth)
	if err != nil {
		return nil
	}
	last := time.Date(end.Year(), end.Month()+1, 0, 0, 0, 0, 0, time.UTC)
	return &last
}

func clampDay(year, month, day int) time.Time {
	last := time.Date(year, time.Month(month)+1, 0, 0, 0, 0, 0, time.UTC).Day()
	if day < 1 {
		day = 1
	}
	if day > last {
		day = last
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func daysBetweenInclusive(start, end time.Time) int {
	if end.Before(start) {
		return 0
	}
	return int(end.Sub(start).Hours()/24) + 1
}

func countWeekdays(start, end time.Time) int {
	count := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		switch d.Weekday() {
		case time.Saturday, time.Sunday:
			continue
		default:
			count++
		}
	}
	return count
}

func countEveryNDays(anchor, rangeStart, rangeEnd time.Time, step int) int {
	if step < 1 || rangeEnd.Before(anchor) {
		return 0
	}
	first := anchor
	if rangeStart.After(anchor) {
		days := int(rangeStart.Sub(anchor).Hours() / 24)
		if days%step == 0 {
			first = rangeStart
		} else {
			first = anchor.AddDate(0, 0, ((days/step)+1)*step)
		}
	}
	if first.After(rangeEnd) {
		return 0
	}
	count := 0
	for d := first; !d.After(rangeEnd); d = d.AddDate(0, 0, step) {
		count++
	}
	return count
}

func round2(value float64) float64 {
	return math.Round(value*100) / 100
}
