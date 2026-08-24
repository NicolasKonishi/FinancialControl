package models

// MonthlyAnalysisRequest is sent from Go to the Python analysis service.
type MonthlyAnalysisRequest struct {
	Year         int           `json:"year"`
	Month        int           `json:"month"`
	Transactions []Transaction `json:"transactions"`
	Categories   []Category    `json:"categories"`
}

// CategoryBreakdown is spending (or income) aggregated by category.
type CategoryBreakdown struct {
	CategoryID   int     `json:"category_id"`
	CategoryName string  `json:"category_name"`
	Total        float64 `json:"total"`
}

// MonthlyAnalysisResponse is returned by the Python analysis service.
type MonthlyAnalysisResponse struct {
	Year             int                 `json:"year"`
	Month            int                 `json:"month"`
	TotalIncome      float64             `json:"total_income"`
	TotalExpense     float64             `json:"total_expense"`
	Balance          float64             `json:"balance"`
	ByCategory       []CategoryBreakdown `json:"by_category"`
	TransactionCount int                 `json:"transaction_count"`
}
