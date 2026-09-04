package server

import (
	"net/http"

	"github.com/NicolasKonishi/FinancialControl/internal/analysis"
	"github.com/NicolasKonishi/FinancialControl/internal/cdi"
	"github.com/NicolasKonishi/FinancialControl/internal/handlers"
	"github.com/NicolasKonishi/FinancialControl/internal/middleware"
	"github.com/NicolasKonishi/FinancialControl/internal/repository"
)

// Dependencies groups the collaborators needed to build HTTP routes.
type Dependencies struct {
	Store          *repository.Store
	AnalysisClient *analysis.Client
	CDI            *cdi.Client
}

// New builds the HTTP handler with routes and middleware.
func New(deps Dependencies) http.Handler {
	categories := &handlers.Categories{Store: deps.Store}
	members := &handlers.Members{Store: deps.Store}
	bills := &handlers.Bills{
		Store:      deps.Store,
		Categories: deps.Store,
		Members:    deps.Store,
		Wallets:    deps.Store,
	}
	transactions := &handlers.Transactions{
		Store:      deps.Store,
		Categories: deps.Store,
		Members:    deps.Store,
	}
	analysisHandler := &handlers.Analysis{
		Transactions: deps.Store,
		Categories:   deps.Store,
		Client:       deps.AnalysisClient,
	}
	forecast := &handlers.Forecast{Store: deps.Store}
	savings := &handlers.Savings{
		Store:   deps.Store,
		Members: deps.Store,
		CDI:     deps.CDI,
	}
	wallets := &handlers.Wallets{
		Store:   deps.Store,
		Members: deps.Store,
	}
	cardInvoices := &handlers.CardInvoices{Store: deps.Store}
	statements := &handlers.Statements{
		Parser:       deps.AnalysisClient,
		Transactions: deps.Store,
		Categories:   deps.Store,
		Members:      deps.Store,
		Wallets:      deps.Store,
		Bills:        deps.Store,
		Imports:      deps.Store,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handlers.Health)

	mux.HandleFunc("/categories", categories.ListOrCreate)
	mux.HandleFunc("GET /categories/{id}", categories.GetByID)
	mux.HandleFunc("PUT /categories/{id}", categories.Update)
	mux.HandleFunc("DELETE /categories/{id}", categories.Delete)

	mux.HandleFunc("/members", members.ListOrCreate)
	mux.HandleFunc("PUT /members/{id}", members.Update)
	mux.HandleFunc("PUT /members/{id}/save-target", members.SetSaveTarget)
	mux.HandleFunc("DELETE /members/{id}/save-target", members.ClearSaveTarget)
	mux.HandleFunc("DELETE /members/{id}", members.Delete)

	mux.HandleFunc("/bills", bills.ListOrCreate)
	mux.HandleFunc("GET /bills/payments", bills.ListPayments)
	mux.HandleFunc("PUT /bills/{id}", bills.Update)
	mux.HandleFunc("PUT /bills/{id}/paid", bills.SetPaid)
	mux.HandleFunc("DELETE /bills/{id}", bills.Delete)

	mux.HandleFunc("/transactions", transactions.ListOrCreate)
	mux.HandleFunc("GET /transactions/{id}", transactions.GetByID)
	mux.HandleFunc("PUT /transactions/{id}", transactions.Update)
	mux.HandleFunc("DELETE /transactions/{id}", transactions.Delete)

	mux.HandleFunc("/savings", savings.ListOrCreate)
	mux.HandleFunc("GET /savings/plan", savings.Plan)
	mux.HandleFunc("GET /savings/months", savings.ListMonthAmounts)
	mux.HandleFunc("PUT /savings/{id}", savings.Update)
	mux.HandleFunc("PUT /savings/{id}/month", savings.SetMonthAmount)
	mux.HandleFunc("PUT /savings/{id}/adjust", savings.Adjust)
	mux.HandleFunc("DELETE /savings/{id}", savings.Delete)

	mux.HandleFunc("/wallets", wallets.ListOrCreate)
	mux.HandleFunc("PUT /wallets/{id}", wallets.Update)
	mux.HandleFunc("DELETE /wallets/{id}", wallets.Delete)
	mux.HandleFunc("POST /wallets/{id}/pay-invoice", wallets.PayInvoice)
	mux.HandleFunc("GET /card-invoices", cardInvoices.List)

	mux.HandleFunc("POST /statements/preview", statements.Preview)
	mux.HandleFunc("POST /statements/import", statements.Import)

	mux.HandleFunc("GET /analysis/monthly", analysisHandler.Monthly)
	mux.HandleFunc("GET /forecast/monthly", forecast.Monthly)

	return middleware.CORS(mux)
}
