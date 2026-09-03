package models

// ParsedStatement is the Python response for a statement PDF/CSV.
type ParsedStatement struct {
	Issuer        string                `json:"issuer"`
	StatementType string                `json:"statement_type"`
	PeriodStart   *string               `json:"period_start"`
	PeriodEnd     *string               `json:"period_end"`
	Balance       *float64              `json:"balance,omitempty"`
	Items         []ParsedStatementItem `json:"items"`
}

// ParsedStatementItem is one line extracted from a statement.
type ParsedStatementItem struct {
	Date          string  `json:"date"`
	Description   string  `json:"description"`
	Amount        float64 `json:"amount"`
	Kind          string  `json:"kind"`
	SuggestedIcon string  `json:"suggested_icon"`
}

// StatementPreview is returned by POST /statements/preview.
type StatementPreview struct {
	Issuer        string                 `json:"issuer"`
	StatementType string                 `json:"statement_type"`
	PeriodStart   *string                `json:"period_start"`
	PeriodEnd     *string                `json:"period_end"`
	Balance       *float64               `json:"balance,omitempty"`
	InvoiceYear   *int                   `json:"invoice_year,omitempty"`
	InvoiceMonth  *int                   `json:"invoice_month,omitempty"`
	ClosingDate   *string                `json:"closing_date,omitempty"`
	DueDate       *string                `json:"due_date,omitempty"`
	WalletID      *int                   `json:"wallet_id,omitempty"`
	MemberID      *int                   `json:"member_id,omitempty"`
	NewCount      int                    `json:"new_count"`
	MatchedCount  int                    `json:"matched_count"`
	SkippedCount  int                    `json:"skipped_count"`
	Items         []StatementPreviewItem `json:"items"`
}

// StatementPreviewItem is a parsed line after matching against the ledger.
type StatementPreviewItem struct {
	Index                int     `json:"index"`
	Date                 string  `json:"date"`
	Description          string  `json:"description"`
	Amount               float64 `json:"amount"`
	Kind                 string  `json:"kind"`
	CategoryID           int     `json:"category_id"`
	SuggestedIcon        string  `json:"suggested_icon"`
	AlreadyRecorded      bool    `json:"already_recorded"`
	MatchedTransactionID *int    `json:"matched_transaction_id,omitempty"`
	Selected             bool    `json:"selected"`
}

// ImportStatementInput is the JSON body for POST /statements/import.
type ImportStatementInput struct {
	WalletID         *int                  `json:"wallet_id"`
	MemberID         *int                  `json:"member_id"`
	ApplyToInvoice   bool                  `json:"apply_to_invoice"`
	StatementType    string                `json:"statement_type"`
	InvoiceYear      *int                  `json:"invoice_year"`
	InvoiceMonth     *int                  `json:"invoice_month"`
	StatementBalance *float64              `json:"statement_balance"`
	PeriodStart      *string               `json:"period_start"`
	PeriodEnd        *string               `json:"period_end"`
	Items            []ImportStatementItem `json:"items"`
}

// ImportStatementItem is one user-confirmed line to persist.
type ImportStatementItem struct {
	Date        string  `json:"date"`
	Description string  `json:"description"`
	Amount      float64 `json:"amount"`
	Type        string  `json:"type"`
	CategoryID  int     `json:"category_id"`
}

// ImportStatementResult is returned after creating the selected lines.
type ImportStatementResult struct {
	Created int           `json:"created"`
	Items   []Transaction `json:"items"`
}
