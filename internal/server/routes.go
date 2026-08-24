package server

import (
	"net/http"

	"github.com/NicolasKonishi/FinancialControl/internal/analysis"
	"github.com/NicolasKonishi/FinancialControl/internal/handlers"
	"github.com/NicolasKonishi/FinancialControl/internal/middleware"
	"github.com/NicolasKonishi/FinancialControl/internal/repository"
)

// Dependencies groups the collaborators needed to build HTTP routes.
type Dependencies struct {
	Store          *repository.Store
	AnalysisClient *analysis.Client
}

// New builds the HTTP handler with routes and middleware.
func New(deps Dependencies) http.Handler {
	categories := &handlers.Categories{Store: deps.Store}
	members := &handlers.Members{Store: deps.Store}
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

	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", handlers.Health)

	mux.HandleFunc("/categories", categories.ListOrCreate)
	mux.HandleFunc("GET /categories/{id}", categories.GetByID)
	mux.HandleFunc("PUT /categories/{id}", categories.Update)
	mux.HandleFunc("DELETE /categories/{id}", categories.Delete)

	mux.HandleFunc("/members", members.ListOrCreate)
	mux.HandleFunc("PUT /members/{id}", members.Update)
	mux.HandleFunc("DELETE /members/{id}", members.Delete)

	mux.HandleFunc("/transactions", transactions.ListOrCreate)
	mux.HandleFunc("GET /transactions/{id}", transactions.GetByID)
	mux.HandleFunc("PUT /transactions/{id}", transactions.Update)
	mux.HandleFunc("DELETE /transactions/{id}", transactions.Delete)

	mux.HandleFunc("GET /analysis/monthly", analysisHandler.Monthly)
	mux.HandleFunc("GET /forecast/monthly", forecast.Monthly)

	return middleware.CORS(mux)
}
