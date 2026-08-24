package models

import "time"

// Bill recurrence kinds.
const (
	BillRecurrenceOngoing = "ongoing" // no end date (internet, electricity…)
	BillRecurrenceUntil   = "until"   // ends on end_month (subscription, credit…)
)

// Bill is a monthly account/bill that may be perpetual or time-bounded.
// MemberIDs lists who shares/pays this bill (1..N family members).
type Bill struct {
	ID         int       `json:"id"`
	Name       string    `json:"name"`
	Amount     float64   `json:"amount"`
	CategoryID int       `json:"category_id"`
	MemberIDs  []int     `json:"member_ids"`
	DueDay     int       `json:"due_day"`
	Recurrence string    `json:"recurrence"`
	StartMonth string    `json:"start_month"` // YYYY-MM
	EndMonth   *string   `json:"end_month,omitempty"`
	Notes      string    `json:"notes,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// CreateBillInput is the JSON body for POST /bills.
type CreateBillInput struct {
	Name       string  `json:"name"`
	Amount     float64 `json:"amount"`
	CategoryID int     `json:"category_id"`
	MemberIDs  []int   `json:"member_ids"`
	DueDay     int     `json:"due_day"`
	Recurrence string  `json:"recurrence"`
	StartMonth string  `json:"start_month"`
	EndMonth   *string `json:"end_month"`
	Notes      string  `json:"notes"`
}

// UpdateBillInput is the JSON body for PUT /bills/{id}.
type UpdateBillInput struct {
	Name       string  `json:"name"`
	Amount     float64 `json:"amount"`
	CategoryID int     `json:"category_id"`
	MemberIDs  []int   `json:"member_ids"`
	DueDay     int     `json:"due_day"`
	Recurrence string  `json:"recurrence"`
	StartMonth string  `json:"start_month"`
	EndMonth   *string `json:"end_month"`
	Notes      string  `json:"notes"`
}

// IsActiveInMonth reports whether the bill charges in year/month.
func (b Bill) IsActiveInMonth(year, month int) bool {
	target := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	start, err := time.Parse("2006-01", b.StartMonth)
	if err != nil {
		return false
	}
	start = time.Date(start.Year(), start.Month(), 1, 0, 0, 0, 0, time.UTC)
	if target.Before(start) {
		return false
	}
	if b.Recurrence == BillRecurrenceOngoing || b.EndMonth == nil {
		return true
	}
	end, err := time.Parse("2006-01", *b.EndMonth)
	if err != nil {
		return false
	}
	end = time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
	return !target.After(end)
}
