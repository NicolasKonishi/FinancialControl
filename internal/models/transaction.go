package models

import "time"

// Transaction types used by the API.
const (
	TransactionTypeIncome  = "income"
	TransactionTypeExpense = "expense"
)

// Transaction is a single income or expense entry.
type Transaction struct {
	ID          int       `json:"id"`
	CategoryID  int       `json:"category_id"`
	MemberID    *int      `json:"member_id,omitempty"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Amount      float64   `json:"amount"`
	Date        time.Time `json:"date"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateTransactionInput is the JSON body for POST /transactions.
type CreateTransactionInput struct {
	CategoryID  int     `json:"category_id"`
	MemberID    *int    `json:"member_id"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"` // YYYY-MM-DD
}

// UpdateTransactionInput is the JSON body for PUT /transactions/{id}.
type UpdateTransactionInput struct {
	CategoryID  int     `json:"category_id"`
	MemberID    *int    `json:"member_id"`
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Date        string  `json:"date"` // YYYY-MM-DD
}
