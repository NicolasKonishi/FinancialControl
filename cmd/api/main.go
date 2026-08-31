package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/analysis"
	"github.com/NicolasKonishi/FinancialControl/internal/cdi"
	"github.com/NicolasKonishi/FinancialControl/internal/config"
	"github.com/NicolasKonishi/FinancialControl/internal/database"
	"github.com/NicolasKonishi/FinancialControl/internal/repository"
	"github.com/NicolasKonishi/FinancialControl/internal/server"
)

func main() {
	cfg := config.Load()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	db, err := database.Connect(ctx, cfg.SQLitePath)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer db.Close()

	handler := server.New(server.Dependencies{
		Store:          repository.NewStore(db),
		AnalysisClient: analysis.NewClient(cfg.PythonAnalysisURL),
		CDI:            cdi.NewClient(),
	})

	httpServer := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		printStartupURLs(cfg)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server failed: %v", err)
		}
	}()

	<-ctx.Done()
	log.Printf("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

func printStartupURLs(cfg config.Config) {
	apiURL := cfg.PublicAPIURL()

	fmt.Println()
	fmt.Println("Fluxo API is running")
	fmt.Println("---------------------")
	fmt.Printf("  Open UI (frontend):  %s\n", cfg.FrontendURL)
	fmt.Printf("  API base URL:        %s\n", apiURL)
	fmt.Printf("  Health check:        %s/health\n", apiURL)
	fmt.Printf("  Categories:          %s/categories\n", apiURL)
	fmt.Printf("  Transactions:        %s/transactions\n", apiURL)
	fmt.Printf("  Monthly forecast:    %s/forecast/monthly\n", apiURL)
	fmt.Printf("  Family members:      %s/members\n", apiURL)
	fmt.Printf("  Monthly bills:       %s/bills\n", apiURL)
	fmt.Printf("  Savings goals:       %s/savings\n", apiURL)
	fmt.Printf("  Python analysis:     %s\n", cfg.PythonAnalysisURL)
	fmt.Printf("  SQLite file:         %s\n", cfg.SQLitePath)
	fmt.Println()
	fmt.Println("Tip: start the UI with:  cd frontend && npm run dev")
	fmt.Println("Tip: start Python with:  cd services/python-analysis && uvicorn main:app --reload --port 8000")
	fmt.Println()
}
