package models

import "time"

// BillPayment records that a bill was marked paid for a given month.
type BillPayment struct {
	BillID int       `json:"bill_id"`
	Year   int       `json:"year"`
	Month  int       `json:"month"`
	PaidAt time.Time `json:"paid_at"`
}

// SetBillPaidInput is the JSON body for PUT /bills/{id}/paid.
type SetBillPaidInput struct {
	Year  int  `json:"year"`
	Month int  `json:"month"`
	Paid  bool `json:"paid"`
}
