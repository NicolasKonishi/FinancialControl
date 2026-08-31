package models

import "time"

// BillPayment records that a bill was marked paid for a given month.
type BillPayment struct {
	BillID         int       `json:"bill_id"`
	Year           int       `json:"year"`
	Month          int       `json:"month"`
	PaidAt         time.Time `json:"paid_at"`
	PaidByMemberID *int      `json:"paid_by_member_id,omitempty"`
	WalletID       *int      `json:"wallet_id,omitempty"`
	Amount         float64   `json:"amount"`
}

// SetBillPaidInput is the JSON body for PUT /bills/{id}/paid.
type SetBillPaidInput struct {
	Year           int  `json:"year"`
	Month          int  `json:"month"`
	Paid           bool `json:"paid"`
	PaidByMemberID *int `json:"paid_by_member_id"`
	WalletID       *int `json:"wallet_id"`
}
