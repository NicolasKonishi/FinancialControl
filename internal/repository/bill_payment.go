package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// ListBillPayments returns payments recorded for year/month.
func (s *Store) ListBillPayments(ctx context.Context, year, month int) ([]models.BillPayment, error) {
	const query = `
		SELECT bill_id, year, month, paid_at
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
		var (
			payment models.BillPayment
			paidAt  string
		)
		if err := rows.Scan(&payment.BillID, &payment.Year, &payment.Month, &paidAt); err != nil {
			return nil, fmt.Errorf("scan bill payment: %w", err)
		}
		parsed, err := parseDateTime(paidAt)
		if err != nil {
			return nil, err
		}
		payment.PaidAt = parsed
		items = append(items, payment)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bill payments: %w", err)
	}
	return items, nil
}

// SetBillPaid marks a bill as paid or unpaid for year/month.
func (s *Store) SetBillPaid(ctx context.Context, billID, year, month int, paid bool) error {
	if _, err := s.GetBillByID(ctx, billID); err != nil {
		return err
	}

	if !paid {
		if _, err := s.db.ExecContext(
			ctx,
			`DELETE FROM bill_payments WHERE bill_id = ? AND year = ? AND month = ?`,
			billID,
			year,
			month,
		); err != nil {
			return fmt.Errorf("clear bill payment: %w", err)
		}
		return nil
	}

	now := formatDateTime(time.Now())
	if _, err := s.db.ExecContext(
		ctx,
		`
			INSERT INTO bill_payments (bill_id, year, month, paid_at)
			VALUES (?, ?, ?, ?)
			ON CONFLICT (bill_id, year, month) DO UPDATE SET paid_at = excluded.paid_at
		`,
		billID,
		year,
		month,
		now,
	); err != nil {
		return fmt.Errorf("set bill payment: %w", err)
	}
	return nil
}
