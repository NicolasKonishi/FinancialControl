package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// ListCardInvoices returns one invoice per credit card for a due month.
func (s *Store) ListCardInvoices(ctx context.Context, year, month int) ([]models.CardInvoice, error) {
	const query = `
		SELECT
			w.id,
			COALESCE(i.amount, 0),
			COALESCE(i.paid_amount, 0),
			COALESCE(i.source, 'calculated'),
			i.statement_period_start,
			i.statement_period_end,
			i.statement_balance,
			i.updated_at,
			w.closing_day,
			w.due_day
		FROM wallets w
		LEFT JOIN card_invoices i
			ON i.wallet_id = w.id AND i.year = ? AND i.month = ?
		WHERE w.kind = 'credit'
		ORDER BY w.id
	`
	rows, err := s.db.QueryContext(ctx, query, year, month)
	if err != nil {
		return nil, fmt.Errorf("list card invoices: %w", err)
	}
	defer rows.Close()

	items := make([]models.CardInvoice, 0)
	for rows.Next() {
		invoice, err := scanCardInvoiceRow(rows, year, month)
		if err != nil {
			return nil, err
		}
		items = append(items, invoice)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate card invoices: %w", err)
	}
	if err := rows.Close(); err != nil {
		return nil, fmt.Errorf("close card invoices: %w", err)
	}
	for i := range items {
		invoice, err := s.withPlannedCardBills(ctx, items[i])
		if err != nil {
			return nil, err
		}
		items[i] = invoice
	}
	return items, nil
}

// GetCardInvoice returns one credit card invoice for a due month.
func (s *Store) GetCardInvoice(ctx context.Context, walletID, year, month int) (models.CardInvoice, error) {
	const query = `
		SELECT
			w.id,
			COALESCE(i.amount, 0),
			COALESCE(i.paid_amount, 0),
			COALESCE(i.source, 'calculated'),
			i.statement_period_start,
			i.statement_period_end,
			i.statement_balance,
			i.updated_at,
			w.closing_day,
			w.due_day
		FROM wallets w
		LEFT JOIN card_invoices i
			ON i.wallet_id = w.id AND i.year = ? AND i.month = ?
		WHERE w.id = ? AND w.kind = 'credit'
	`
	invoice, err := scanCardInvoiceRow(s.db.QueryRowContext(ctx, query, year, month, walletID), year, month)
	if errors.Is(err, sql.ErrNoRows) {
		return models.CardInvoice{}, ErrNotFound
	}
	if err != nil {
		return models.CardInvoice{}, err
	}
	return s.withPlannedCardBills(ctx, invoice)
}

// ReconcileCardInvoice records the authoritative total reported by a statement.
func (s *Store) ReconcileCardInvoice(
	ctx context.Context,
	walletID, year, month int,
	amount float64,
	periodStart, periodEnd *string,
) (models.CardInvoice, error) {
	wallet, err := s.GetWalletByID(ctx, walletID)
	if err != nil {
		return models.CardInvoice{}, err
	}
	if !models.IsCredit(wallet.Kind) {
		return models.CardInvoice{}, ErrNotCredit
	}
	amount = math.Round(math.Abs(amount)*100) / 100

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.CardInvoice{}, fmt.Errorf("begin reconcile card invoice: %w", err)
	}
	defer tx.Rollback()

	now := formatDateTime(time.Now())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO card_invoices (
			wallet_id, year, month, amount, paid_amount, source,
			statement_period_start, statement_period_end, statement_balance, updated_at
		) VALUES (?, ?, ?, ?, 0, 'statement', ?, ?, ?, ?)
		ON CONFLICT (wallet_id, year, month) DO UPDATE SET
			amount = excluded.amount,
			source = 'statement',
			statement_period_start = excluded.statement_period_start,
			statement_period_end = excluded.statement_period_end,
			statement_balance = excluded.statement_balance,
			updated_at = excluded.updated_at
	`, walletID, year, month, amount, nullString(periodStart), nullString(periodEnd), amount, now); err != nil {
		return models.CardInvoice{}, fmt.Errorf("reconcile card invoice: %w", err)
	}
	if err := syncWalletInvoiceBalanceTx(ctx, tx, walletID); err != nil {
		return models.CardInvoice{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.CardInvoice{}, fmt.Errorf("commit reconcile card invoice: %w", err)
	}
	return s.GetCardInvoice(ctx, walletID, year, month)
}

// PayCardInvoice debits a cash wallet and records payment against one due month.
func (s *Store) PayCardInvoice(ctx context.Context, creditWalletID int, input models.PayInvoiceInput) (models.CardInvoice, error) {
	amount := math.Round(input.Amount*100) / 100
	if !(amount > 0) || input.Year < 2000 || input.Month < 1 || input.Month > 12 {
		return models.CardInvoice{}, ErrInvalidAmount
	}
	if input.FromWalletID < 1 || input.FromWalletID == creditWalletID {
		return models.CardInvoice{}, ErrInvalidAmount
	}

	invoice, err := s.GetCardInvoice(ctx, creditWalletID, input.Year, input.Month)
	if err != nil {
		return models.CardInvoice{}, err
	}
	if invoice.Outstanding <= 0 {
		return models.CardInvoice{}, ErrInvoiceEmpty
	}
	if amount > invoice.Outstanding {
		amount = invoice.Outstanding
	}
	from, err := s.GetWalletByID(ctx, input.FromWalletID)
	if err != nil {
		return models.CardInvoice{}, err
	}
	if models.IsCredit(from.Kind) {
		return models.CardInvoice{}, ErrNotCredit
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.CardInvoice{}, fmt.Errorf("begin pay card invoice: %w", err)
	}
	defer tx.Rollback()
	if err := adjustWalletTx(ctx, tx, from.ID, -amount); err != nil {
		return models.CardInvoice{}, err
	}
	now := formatDateTime(time.Now())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO card_invoices (wallet_id, year, month, amount, paid_amount, source, updated_at)
		VALUES (?, ?, ?, 0, 0, 'calculated', ?)
		ON CONFLICT (wallet_id, year, month) DO NOTHING
	`, creditWalletID, input.Year, input.Month, now); err != nil {
		return models.CardInvoice{}, fmt.Errorf("ensure card invoice payment row: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE card_invoices
		SET paid_amount = paid_amount + ?, updated_at = ?
		WHERE wallet_id = ? AND year = ? AND month = ?
	`, amount, now, creditWalletID, input.Year, input.Month); err != nil {
		return models.CardInvoice{}, fmt.Errorf("pay card invoice: %w", err)
	}
	if amount >= invoice.Outstanding {
		if err := s.markCardBillsPaidTx(ctx, tx, creditWalletID, input.Year, input.Month, now); err != nil {
			return models.CardInvoice{}, err
		}
	}
	if err := syncWalletInvoiceBalanceTx(ctx, tx, creditWalletID); err != nil {
		return models.CardInvoice{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.CardInvoice{}, fmt.Errorf("commit pay card invoice: %w", err)
	}
	return s.GetCardInvoice(ctx, creditWalletID, input.Year, input.Month)
}

func (s *Store) withPlannedCardBills(ctx context.Context, invoice models.CardInvoice) (models.CardInvoice, error) {
	if invoice.Source == "statement" {
		return models.FinalizeCardInvoice(invoice), nil
	}
	bills, err := s.ListBills(ctx)
	if err != nil {
		return models.CardInvoice{}, err
	}
	for _, bill := range bills {
		if bill.WalletID == nil || *bill.WalletID != invoice.WalletID {
			continue
		}
		invoice.Amount += bill.ChargeForMonth(invoice.Year, invoice.Month)
	}
	return models.FinalizeCardInvoice(invoice), nil
}

func (s *Store) markCardBillsPaidTx(ctx context.Context, tx *sql.Tx, walletID, year, month int, paidAt string) error {
	rows, err := tx.QueryContext(ctx, `
		SELECT b.id, b.member_id
		FROM bills b
		WHERE b.wallet_id = ?
	`, walletID)
	if err != nil {
		return fmt.Errorf("list card bill components: %w", err)
	}
	type component struct {
		id       int
		memberID sql.NullInt64
	}
	components := make([]component, 0)
	for rows.Next() {
		var item component
		if err := rows.Scan(&item.id, &item.memberID); err != nil {
			rows.Close()
			return fmt.Errorf("scan card bill component: %w", err)
		}
		components = append(components, item)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close card bill components: %w", err)
	}

	var ownerID sql.NullInt64
	if err := tx.QueryRowContext(ctx, `SELECT member_id FROM wallets WHERE id = ?`, walletID).Scan(&ownerID); err != nil {
		return fmt.Errorf("get card owner: %w", err)
	}
	for _, item := range components {
		bill, err := getBillByIDTx(ctx, tx, item.id)
		if err != nil {
			return err
		}
		if !bill.IsActiveInMonth(year, month) {
			continue
		}
		payer := ownerID
		if !payer.Valid && item.memberID.Valid {
			payer = item.memberID
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO bill_payments (bill_id, year, month, paid_at, paid_by_member_id, wallet_id, amount)
			VALUES (?, ?, ?, ?, ?, ?, 0)
			ON CONFLICT (bill_id, year, month) DO UPDATE SET
				paid_at = excluded.paid_at,
				paid_by_member_id = excluded.paid_by_member_id,
				wallet_id = excluded.wallet_id,
				amount = 0
		`, item.id, year, month, paidAt, nullableInt64(payer), walletID); err != nil {
			return fmt.Errorf("mark card bill component paid: %w", err)
		}
	}
	return nil
}

func getBillByIDTx(ctx context.Context, tx *sql.Tx, id int) (models.Bill, error) {
	var (
		bill     models.Bill
		endMonth sql.NullString
		walletID sql.NullInt64
		created  string
	)
	if err := tx.QueryRowContext(ctx, `
		SELECT id, name, amount, amount_mode, interest_rate, category_id, due_day, frequency,
			recurrence, start_month, end_month, notes, created_at, wallet_id
		FROM bills WHERE id = ?
	`, id).Scan(
		&bill.ID, &bill.Name, &bill.Amount, &bill.AmountMode, &bill.InterestRate,
		&bill.CategoryID, &bill.DueDay, &bill.Frequency, &bill.Recurrence,
		&bill.StartMonth, &endMonth, &bill.Notes, &created, &walletID,
	); err != nil {
		return models.Bill{}, fmt.Errorf("get card bill component: %w", err)
	}
	if endMonth.Valid {
		value := endMonth.String
		bill.EndMonth = &value
	}
	bill.WalletID = fromNullInt(walletID)
	return bill, nil
}

func nullableInt64(value sql.NullInt64) any {
	if !value.Valid {
		return nil
	}
	return value.Int64
}

func adjustCardInvoiceTx(ctx context.Context, tx *sql.Tx, wallet models.Wallet, at time.Time, delta float64) error {
	cycle := models.CardCycleForPurchase(wallet, at)
	now := formatDateTime(time.Now())
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO card_invoices (wallet_id, year, month, amount, paid_amount, source, updated_at)
		VALUES (?, ?, ?, 0, 0, 'calculated', ?)
		ON CONFLICT (wallet_id, year, month) DO NOTHING
	`, wallet.ID, cycle.Year, cycle.Month, now); err != nil {
		return fmt.Errorf("adjust card invoice: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `
		UPDATE card_invoices
		SET amount = MAX(0, amount + ?), updated_at = ?
		WHERE wallet_id = ? AND year = ? AND month = ?
	`, delta, now, wallet.ID, cycle.Year, cycle.Month); err != nil {
		return fmt.Errorf("adjust card invoice amount: %w", err)
	}
	return syncWalletInvoiceBalanceTx(ctx, tx, wallet.ID)
}

func syncWalletInvoiceBalanceTx(ctx context.Context, tx *sql.Tx, walletID int) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE wallets
		SET invoice_balance = COALESCE((
			SELECT ROUND(SUM(MAX(amount - paid_amount, 0)), 2)
			FROM card_invoices
			WHERE wallet_id = ?
		), 0)
		WHERE id = ?
	`, walletID, walletID); err != nil {
		return fmt.Errorf("sync card invoice balance: %w", err)
	}
	return nil
}

func scanCardInvoiceRow(row interface{ Scan(...any) error }, year, month int) (models.CardInvoice, error) {
	var (
		invoice                         models.CardInvoice
		periodStart, periodEnd, updated sql.NullString
		statementBalance                sql.NullFloat64
		closingDay, dueDay              sql.NullInt64
	)
	if err := row.Scan(
		&invoice.WalletID,
		&invoice.Amount,
		&invoice.PaidAmount,
		&invoice.Source,
		&periodStart,
		&periodEnd,
		&statementBalance,
		&updated,
		&closingDay,
		&dueDay,
	); err != nil {
		return models.CardInvoice{}, err
	}
	invoice.Year = year
	invoice.Month = month
	wallet := models.Wallet{ClosingDay: fromNullInt(closingDay), DueDay: fromNullInt(dueDay)}
	cycle := models.CardCycleForDueMonth(wallet, year, month)
	invoice.ClosingDate = formatDate(cycle.ClosingDate)
	invoice.DueDate = formatDate(cycle.DueDate)
	invoice.StatementPeriodStart = fromNullString(periodStart)
	invoice.StatementPeriodEnd = fromNullString(periodEnd)
	if statementBalance.Valid {
		value := statementBalance.Float64
		invoice.StatementBalance = &value
	}
	if updated.Valid {
		value, err := parseDateTime(updated.String)
		if err != nil {
			return models.CardInvoice{}, err
		}
		invoice.UpdatedAt = &value
	}
	return models.FinalizeCardInvoice(invoice), nil
}

func fromNullString(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	result := value.String
	return &result
}
