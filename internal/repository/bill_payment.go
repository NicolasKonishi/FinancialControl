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

// ListBillPayments returns payments recorded for year/month.
func (s *Store) ListBillPayments(ctx context.Context, year, month int) ([]models.BillPayment, error) {
	const query = `
		SELECT bill_id, year, month, paid_at, paid_by_member_id, wallet_id, amount
		FROM bill_payments
		WHERE year = ? AND month = ?
		ORDER BY bill_id ASC
	`

	rows, err := s.db.QueryContext(ctx, query, year, month)
	if err != nil {
		return nil, fmt.Errorf("query bill payments: %w", err)
	}
	defer rows.Close()

	items := make([]models.BillPayment, 0)
	for rows.Next() {
		payment, err := scanBillPayment(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bill payments: %w", err)
	}
	return items, nil
}

// SetBillPaid marks a bill as paid or unpaid for year/month and moves wallet cash.
func (s *Store) SetBillPaid(ctx context.Context, billID, year, month int, paid bool, paidByMemberID, walletID *int) error {
	bill, err := s.GetBillByID(ctx, billID)
	if err != nil {
		return err
	}

	existing, err := s.getBillPayment(ctx, billID, year, month)
	if err != nil && !errors.Is(err, ErrNotFound) {
		return err
	}
	hasExisting := err == nil

	var wallet models.Wallet
	if paid {
		if paidByMemberID == nil || *paidByMemberID < 1 {
			return fmt.Errorf("paid_by_member_id is required")
		}
		wallet, err = s.resolvePayerWallet(ctx, *paidByMemberID, walletID)
		if err != nil {
			return err
		}
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin bill payment: %w", err)
	}
	defer tx.Rollback()

	if hasExisting && existing.WalletID != nil && existing.Amount != 0 {
		if err := adjustWalletTx(ctx, tx, *existing.WalletID, existing.Amount); err != nil {
			return err
		}
	}

	if !paid {
		if _, err := tx.ExecContext(
			ctx,
			`DELETE FROM bill_payments WHERE bill_id = ? AND year = ? AND month = ?`,
			billID,
			year,
			month,
		); err != nil {
			return fmt.Errorf("clear bill payment: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit clear bill payment: %w", err)
		}
		return nil
	}

	amount := bill.ChargeForMonth(year, month)
	if amount < 0 {
		amount = 0
	}
	amount = math.Round(amount*100) / 100

	if err := adjustWalletTx(ctx, tx, wallet.ID, -amount); err != nil {
		return err
	}

	now := formatDateTime(time.Now())
	if _, err := tx.ExecContext(
		ctx,
		`
			INSERT INTO bill_payments (bill_id, year, month, paid_at, paid_by_member_id, wallet_id, amount)
			VALUES (?, ?, ?, ?, ?, ?, ?)
			ON CONFLICT (bill_id, year, month) DO UPDATE SET
				paid_at = excluded.paid_at,
				paid_by_member_id = excluded.paid_by_member_id,
				wallet_id = excluded.wallet_id,
				amount = excluded.amount
		`,
		billID,
		year,
		month,
		now,
		*paidByMemberID,
		wallet.ID,
		amount,
	); err != nil {
		return fmt.Errorf("set bill payment: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit bill payment: %w", err)
	}
	return nil
}

func (s *Store) getBillPayment(ctx context.Context, billID, year, month int) (models.BillPayment, error) {
	const query = `
		SELECT bill_id, year, month, paid_at, paid_by_member_id, wallet_id, amount
		FROM bill_payments
		WHERE bill_id = ? AND year = ? AND month = ?
	`

	payment, err := scanBillPayment(s.db.QueryRowContext(ctx, query, billID, year, month))
	if errors.Is(err, sql.ErrNoRows) {
		return models.BillPayment{}, ErrNotFound
	}
	if err != nil {
		return models.BillPayment{}, fmt.Errorf("get bill payment: %w", err)
	}
	return payment, nil
}

func (s *Store) resolvePayerWallet(ctx context.Context, memberID int, walletID *int) (models.Wallet, error) {
	if walletID != nil && *walletID > 0 {
		wallet, err := s.GetWalletByID(ctx, *walletID)
		if err != nil {
			return models.Wallet{}, err
		}
		if wallet.MemberID == nil || *wallet.MemberID != memberID {
			return models.Wallet{}, ErrWalletOwner
		}
		return wallet, nil
	}

	const query = `
		SELECT id, name, kind, member_id, balance, created_at
		FROM wallets
		WHERE member_id = ?
		ORDER BY id ASC
	`

	rows, err := s.db.QueryContext(ctx, query, memberID)
	if err != nil {
		return models.Wallet{}, fmt.Errorf("list member wallets: %w", err)
	}
	defer rows.Close()

	items := make([]models.Wallet, 0)
	for rows.Next() {
		wallet, err := scanWallet(rows)
		if err != nil {
			return models.Wallet{}, err
		}
		items = append(items, wallet)
	}
	if err := rows.Err(); err != nil {
		return models.Wallet{}, fmt.Errorf("iterate member wallets: %w", err)
	}

	wallet, ok := models.PreferredWallet(items)
	if !ok {
		return models.Wallet{}, ErrNoWallet
	}
	return wallet, nil
}

func adjustWalletTx(ctx context.Context, tx *sql.Tx, walletID int, delta float64) error {
	var balance float64
	err := tx.QueryRowContext(ctx, `SELECT balance FROM wallets WHERE id = ?`, walletID).Scan(&balance)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("get wallet balance: %w", err)
	}
	next := math.Round((balance+delta)*100) / 100
	if _, err := tx.ExecContext(ctx, `UPDATE wallets SET balance = ? WHERE id = ?`, next, walletID); err != nil {
		return fmt.Errorf("adjust wallet: %w", err)
	}
	return nil
}

func scanBillPayment(row interface {
	Scan(dest ...any) error
}) (models.BillPayment, error) {
	var (
		payment        models.BillPayment
		paidAt         string
		paidByMemberID sql.NullInt64
		walletID       sql.NullInt64
	)
	if err := row.Scan(
		&payment.BillID,
		&payment.Year,
		&payment.Month,
		&paidAt,
		&paidByMemberID,
		&walletID,
		&payment.Amount,
	); err != nil {
		return models.BillPayment{}, err
	}
	parsed, err := parseDateTime(paidAt)
	if err != nil {
		return models.BillPayment{}, err
	}
	payment.PaidAt = parsed
	payment.PaidByMemberID = fromNullInt(paidByMemberID)
	payment.WalletID = fromNullInt(walletID)
	return payment, nil
}
