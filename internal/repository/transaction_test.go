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
			closing_day INTEGER,
			due_day INTEGER,
			credit_limit REAL NOT NULL DEFAULT 0,
			invoice_balance REAL NOT NULL DEFAULT 0,
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
		CREATE TABLE card_invoices (
			wallet_id INTEGER NOT NULL,
			year INTEGER NOT NULL,
			month INTEGER NOT NULL,
			amount REAL NOT NULL DEFAULT 0,
			paid_amount REAL NOT NULL DEFAULT 0,
			source TEXT NOT NULL DEFAULT 'calculated',
			statement_period_start TEXT,
			statement_period_end TEXT,
			statement_balance REAL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (wallet_id, year, month)
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

func TestCreditCardExpenseRaisesInvoiceAndPayInvoiceDebitsChecking(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, "file:txcredit?mode=memory&cache=shared")
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
			closing_day INTEGER,
			due_day INTEGER,
			credit_limit REAL NOT NULL DEFAULT 0,
			invoice_balance REAL NOT NULL DEFAULT 0,
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
		CREATE TABLE card_invoices (
			wallet_id INTEGER NOT NULL,
			year INTEGER NOT NULL,
			month INTEGER NOT NULL,
			amount REAL NOT NULL DEFAULT 0,
			paid_amount REAL NOT NULL DEFAULT 0,
			source TEXT NOT NULL DEFAULT 'calculated',
			statement_period_start TEXT,
			statement_period_end TEXT,
			statement_balance REAL,
			updated_at TEXT NOT NULL,
			PRIMARY KEY (wallet_id, year, month)
		);
		CREATE TABLE bills (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			amount REAL NOT NULL,
			amount_mode TEXT NOT NULL DEFAULT 'fixed',
			interest_rate REAL NOT NULL DEFAULT 0,
			category_id INTEGER NOT NULL,
			member_id INTEGER,
			due_day INTEGER NOT NULL,
			frequency TEXT NOT NULL DEFAULT 'monthly',
			recurrence TEXT NOT NULL,
			start_month TEXT NOT NULL,
			end_month TEXT,
			notes TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			wallet_id INTEGER,
			source TEXT NOT NULL DEFAULT 'manual',
			installment_start INTEGER NOT NULL DEFAULT 0,
			installment_total INTEGER NOT NULL DEFAULT 0
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
	closing, due := 10, 17
	card, err := store.CreateWallet(ctx, models.CreateWalletInput{
		Name:        "Nubank",
		Kind:        models.WalletCredit,
		MemberID:    &member.ID,
		ClosingDay:  &closing,
		DueDay:      &due,
		CreditLimit: 5000,
	})
	if err != nil {
		t.Fatalf("card: %v", err)
	}
	checking, err := store.CreateWallet(ctx, models.CreateWalletInput{
		Name:     "Pix",
		Kind:     models.WalletChecking,
		MemberID: &member.ID,
		Balance:  1000,
	})
	if err != nil {
		t.Fatalf("checking: %v", err)
	}

	_, err = store.CreateTransaction(ctx, models.Transaction{
		CategoryID:  category.ID,
		MemberID:    &member.ID,
		WalletID:    &card.ID,
		Type:        models.TransactionTypeExpense,
		Description: "Mercado",
		Amount:      200,
		Date:        time.Date(2026, 8, 15, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("create transaction: %v", err)
	}

	card, err = store.GetWalletByID(ctx, card.ID)
	if err != nil {
		t.Fatalf("get card: %v", err)
	}
	if card.InvoiceBalance != 200 {
		t.Fatalf("invoice = %v, want 200", card.InvoiceBalance)
	}
	if card.Balance != 0 {
		t.Fatalf("credit cash balance = %v, want 0", card.Balance)
	}

	paid, err := store.PayWalletInvoice(ctx, card.ID, models.PayInvoiceInput{
		Amount:       80,
		FromWalletID: checking.ID,
		Year:         2026,
		Month:        9,
	})
	if err != nil {
		t.Fatalf("pay invoice: %v", err)
	}
	if paid.InvoiceBalance != 120 {
		t.Fatalf("invoice after pay = %v, want 120", paid.InvoiceBalance)
	}
	checking, err = store.GetWalletByID(ctx, checking.ID)
	if err != nil {
		t.Fatalf("get checking: %v", err)
	}
	if checking.Balance != 920 {
		t.Fatalf("checking after pay = %v, want 920", checking.Balance)
	}
}

func TestPayCardInvoiceMarksLinkedBills(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	db, err := database.Connect(ctx, "file:paycardbills?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	schema := `
		CREATE TABLE members (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, monthly_salary REAL NOT NULL DEFAULT 0, created_at TEXT NOT NULL);
		CREATE TABLE categories (id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, description TEXT NOT NULL DEFAULT '', icon TEXT NOT NULL DEFAULT 'other', created_at TEXT NOT NULL);
		CREATE TABLE wallets (
			id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, kind TEXT NOT NULL,
			member_id INTEGER REFERENCES members(id), balance REAL NOT NULL DEFAULT 0,
			closing_day INTEGER, due_day INTEGER, credit_limit REAL NOT NULL DEFAULT 0,
			invoice_balance REAL NOT NULL DEFAULT 0, created_at TEXT NOT NULL
		);
		CREATE TABLE transactions (
			id INTEGER PRIMARY KEY AUTOINCREMENT, category_id INTEGER NOT NULL, member_id INTEGER,
			wallet_id INTEGER, type TEXT NOT NULL, description TEXT NOT NULL, amount REAL NOT NULL,
			date TEXT NOT NULL, created_at TEXT NOT NULL
		);
		CREATE TABLE card_invoices (
			wallet_id INTEGER NOT NULL, year INTEGER NOT NULL, month INTEGER NOT NULL,
			amount REAL NOT NULL DEFAULT 0, paid_amount REAL NOT NULL DEFAULT 0,
			source TEXT NOT NULL DEFAULT 'calculated', statement_period_start TEXT,
			statement_period_end TEXT, statement_balance REAL, updated_at TEXT NOT NULL,
			PRIMARY KEY (wallet_id, year, month)
		);
		CREATE TABLE bills (
			id INTEGER PRIMARY KEY AUTOINCREMENT, name TEXT NOT NULL, amount REAL NOT NULL,
			amount_mode TEXT NOT NULL DEFAULT 'fixed', interest_rate REAL NOT NULL DEFAULT 0,
			category_id INTEGER NOT NULL, member_id INTEGER, due_day INTEGER NOT NULL,
			frequency TEXT NOT NULL DEFAULT 'monthly', recurrence TEXT NOT NULL,
			start_month TEXT NOT NULL, end_month TEXT, notes TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL, wallet_id INTEGER, source TEXT NOT NULL DEFAULT 'manual',
			installment_start INTEGER NOT NULL DEFAULT 0, installment_total INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE bill_members (bill_id INTEGER NOT NULL, member_id INTEGER NOT NULL, PRIMARY KEY (bill_id, member_id));
		CREATE TABLE bill_payments (
			bill_id INTEGER NOT NULL, year INTEGER NOT NULL, month INTEGER NOT NULL,
			paid_at TEXT NOT NULL, paid_by_member_id INTEGER, wallet_id INTEGER, amount REAL NOT NULL DEFAULT 0,
			PRIMARY KEY (bill_id, year, month)
		);
	`
	if _, err := db.ExecContext(ctx, schema); err != nil {
		t.Fatalf("schema: %v", err)
	}

	store := NewStore(db)
	member, _ := store.CreateMember(ctx, models.CreateMemberInput{Name: "Ana", MonthlySalary: 3000})
	category, _ := store.CreateCategory(ctx, models.CreateCategoryInput{Name: "Assinaturas", Icon: "subscriptions"})
	closing, due := 14, 21
	card, err := store.CreateWallet(ctx, models.CreateWalletInput{
		Name: "Nubank", Kind: models.WalletCredit, MemberID: &member.ID, ClosingDay: &closing, DueDay: &due, CreditLimit: 5000,
	})
	if err != nil {
		t.Fatalf("card: %v", err)
	}
	checking, err := store.CreateWallet(ctx, models.CreateWalletInput{
		Name: "Pix", Kind: models.WalletChecking, MemberID: &member.ID, Balance: 1000,
	})
	if err != nil {
		t.Fatalf("checking: %v", err)
	}
	bill, err := store.CreateBill(ctx, models.Bill{
		Name: "Nu Seguro Celular", Amount: 12.9, CategoryID: category.ID, MemberIDs: []int{member.ID},
		WalletID: &card.ID, DueDay: 21, Frequency: models.BillFrequencyMonthly,
		Recurrence: models.BillRecurrenceOngoing, StartMonth: "2026-09", Source: models.BillSourceManual,
	})
	if err != nil {
		t.Fatalf("bill: %v", err)
	}
	_, err = store.CreateTransaction(ctx, models.Transaction{
		CategoryID: category.ID, MemberID: &member.ID, WalletID: &card.ID,
		Type: models.TransactionTypeExpense, Description: "Mercado", Amount: 200,
		Date: time.Date(2026, 8, 20, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("tx: %v", err)
	}

	if _, err := store.PayWalletInvoice(ctx, card.ID, models.PayInvoiceInput{
		Amount: 200, FromWalletID: checking.ID, Year: 2026, Month: 9,
	}); err != nil {
		t.Fatalf("pay invoice: %v", err)
	}
	payments, err := store.ListBillPayments(ctx, 2026, 9)
	if err != nil {
		t.Fatal(err)
	}
	if len(payments) != 1 || payments[0].BillID != bill.ID {
		t.Fatalf("invoice payment should check linked bills, got %+v", payments)
	}
}
