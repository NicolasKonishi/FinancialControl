package repository

import (
	"context"
	"testing"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/database"
	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

func TestCreateTransactionDebitsWalletWithoutDeadlock(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, "file:txwallet?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
		CREATE TABLE members (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			monthly_salary REAL NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);
		CREATE TABLE categories (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			icon TEXT NOT NULL DEFAULT 'other',
			created_at TEXT NOT NULL
		);
		CREATE TABLE wallets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			kind TEXT NOT NULL,
			member_id INTEGER REFERENCES members(id),
			balance REAL NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL
		);
		CREATE TABLE transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category_id INTEGER NOT NULL REFERENCES categories(id),
			member_id INTEGER REFERENCES members(id),
			wallet_id INTEGER REFERENCES wallets(id),
			type TEXT NOT NULL,
			description TEXT NOT NULL,
			amount REAL NOT NULL,
			date TEXT NOT NULL,
			created_at TEXT NOT NULL
		);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		t.Fatalf("schema: %v", err)
	}

	store := NewStore(db)
	member, err := store.CreateMember(ctx, models.CreateMemberInput{Name: "Ana", MonthlySalary: 3000})
	if err != nil {
		t.Fatalf("member: %v", err)
	}
	category, err := store.CreateCategory(ctx, models.CreateCategoryInput{Name: "Mercado", Icon: "market"})
	if err != nil {
		t.Fatalf("category: %v", err)
	}
	wallet, err := store.CreateWallet(ctx, models.CreateWalletInput{
		Name:     "Conta",
		Kind:     models.WalletChecking,
		MemberID: &member.ID,
		Balance:  500,
	})
	if err != nil {
		t.Fatalf("wallet: %v", err)
	}

	created, err := store.CreateTransaction(ctx, models.Transaction{
		CategoryID:  category.ID,
		MemberID:    &member.ID,
		WalletID:    &wallet.ID,
		Type:        models.TransactionTypeExpense,
		Description: "Mercado",
		Amount:      80,
		Date:        time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}
	if created.ID < 1 {
		t.Fatalf("expected persisted id, got %d", created.ID)
	}

	updated, err := store.GetWalletByID(ctx, wallet.ID)
	if err != nil {
		t.Fatalf("get wallet: %v", err)
	}
	if updated.Balance != 420 {
		t.Fatalf("wallet after expense = %v, want 420", updated.Balance)
	}
}
