package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// CreateBill inserts a monthly bill and its paying members.
func (s *Store) CreateBill(ctx context.Context, bill models.Bill) (models.Bill, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Bill{}, fmt.Errorf("begin create bill: %w", err)
	}
	defer tx.Rollback()

	const query = `
		INSERT INTO bills (
			name, amount, amount_mode, interest_rate, category_id, member_id, due_day, frequency,
			recurrence, start_month, end_month, notes, created_at
		) VALUES (?, ?, ?, ?, ?, NULL, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, name, amount, amount_mode, interest_rate, category_id, due_day, frequency,
			recurrence, start_month, end_month, notes, created_at
	`

	now := formatDateTime(time.Now())
	var (
		created   models.Bill
		endMonth  sql.NullString
		createdAt string
	)
	err = tx.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(bill.Name),
		bill.Amount,
		models.NormalizeAmountMode(bill.AmountMode),
		bill.InterestRate,
		bill.CategoryID,
		bill.DueDay,
		models.NormalizeFrequency(bill.Frequency),
		bill.Recurrence,
		bill.StartMonth,
		nullString(bill.EndMonth),
		strings.TrimSpace(bill.Notes),
		now,
	).Scan(
		&created.ID,
		&created.Name,
		&created.Amount,
		&created.AmountMode,
		&created.InterestRate,
		&created.CategoryID,
		&created.DueDay,
		&created.Frequency,
		&created.Recurrence,
		&created.StartMonth,
		&endMonth,
		&created.Notes,
		&createdAt,
	)
	if err != nil {
		return models.Bill{}, fmt.Errorf("create bill: %w", err)
	}

	if err := replaceBillMembersTx(ctx, tx, created.ID, bill.MemberIDs); err != nil {
		return models.Bill{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.Bill{}, fmt.Errorf("commit create bill: %w", err)
	}

	created.MemberIDs = normalizeMemberIDs(bill.MemberIDs)
	if endMonth.Valid {
		value := endMonth.String
		created.EndMonth = &value
	}
	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.Bill{}, err
	}
	created.CreatedAt = parsed
	return created, nil
}

// ListBills returns all bills ordered by due_day, then name.
func (s *Store) ListBills(ctx context.Context) ([]models.Bill, error) {
	const query = `
		SELECT id, name, amount, amount_mode, interest_rate, category_id, due_day, frequency,
			recurrence, start_month, end_month, notes, created_at
		FROM bills
		ORDER BY due_day ASC, name ASC
	`
	return s.scanBills(ctx, query)
}

// ListBillsActiveInMonth returns bills that charge in year/month.
func (s *Store) ListBillsActiveInMonth(ctx context.Context, year, month int) ([]models.Bill, error) {
	all, err := s.ListBills(ctx)
	if err != nil {
		return nil, err
	}
	active := make([]models.Bill, 0)
	for _, bill := range all {
		if bill.IsActiveInMonth(year, month) {
			active = append(active, bill)
		}
	}
	return active, nil
}

// GetBillByID returns one bill or ErrNotFound.
func (s *Store) GetBillByID(ctx context.Context, id int) (models.Bill, error) {
	const query = `
		SELECT id, name, amount, amount_mode, interest_rate, category_id, due_day, frequency,
			recurrence, start_month, end_month, notes, created_at
		FROM bills
		WHERE id = ?
	`

	var (
		bill      models.Bill
		endMonth  sql.NullString
		createdAt string
	)
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&bill.ID,
		&bill.Name,
		&bill.Amount,
		&bill.AmountMode,
		&bill.InterestRate,
		&bill.CategoryID,
		&bill.DueDay,
		&bill.Frequency,
		&bill.Recurrence,
		&bill.StartMonth,
		&endMonth,
		&bill.Notes,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Bill{}, ErrNotFound
	}
	if err != nil {
		return models.Bill{}, fmt.Errorf("get bill: %w", err)
	}

	members, err := s.listBillMemberIDs(ctx, id)
	if err != nil {
		return models.Bill{}, err
	}
	bill.MemberIDs = members
	if endMonth.Valid {
		value := endMonth.String
		bill.EndMonth = &value
	}
	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.Bill{}, err
	}
	bill.CreatedAt = parsed
	return bill, nil
}

// UpdateBill replaces an existing bill and its paying members.
func (s *Store) UpdateBill(ctx context.Context, id int, bill models.Bill) (models.Bill, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Bill{}, fmt.Errorf("begin update bill: %w", err)
	}
	defer tx.Rollback()

	const query = `
		UPDATE bills
		SET name = ?, amount = ?, amount_mode = ?, interest_rate = ?, category_id = ?, member_id = NULL, due_day = ?,
			frequency = ?, recurrence = ?, start_month = ?, end_month = ?, notes = ?
		WHERE id = ?
		RETURNING id, name, amount, amount_mode, interest_rate, category_id, due_day, frequency,
			recurrence, start_month, end_month, notes, created_at
	`

	var (
		updated   models.Bill
		endMonth  sql.NullString
		createdAt string
	)
	err = tx.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(bill.Name),
		bill.Amount,
		models.NormalizeAmountMode(bill.AmountMode),
		bill.InterestRate,
		bill.CategoryID,
		bill.DueDay,
		models.NormalizeFrequency(bill.Frequency),
		bill.Recurrence,
		bill.StartMonth,
		nullString(bill.EndMonth),
		strings.TrimSpace(bill.Notes),
		id,
	).Scan(
		&updated.ID,
		&updated.Name,
		&updated.Amount,
		&updated.AmountMode,
		&updated.InterestRate,
		&updated.CategoryID,
		&updated.DueDay,
		&updated.Frequency,
		&updated.Recurrence,
		&updated.StartMonth,
		&endMonth,
		&updated.Notes,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Bill{}, ErrNotFound
	}
	if err != nil {
		return models.Bill{}, fmt.Errorf("update bill: %w", err)
	}

	if err := replaceBillMembersTx(ctx, tx, id, bill.MemberIDs); err != nil {
		return models.Bill{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.Bill{}, fmt.Errorf("commit update bill: %w", err)
	}

	updated.MemberIDs = normalizeMemberIDs(bill.MemberIDs)
	if endMonth.Valid {
		value := endMonth.String
		updated.EndMonth = &value
	}
	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.Bill{}, err
	}
	updated.CreatedAt = parsed
	return updated, nil
}

// DeleteBill removes a bill by id.
func (s *Store) DeleteBill(ctx context.Context, id int) error {
	const query = `DELETE FROM bills WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete bill: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete bill rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) scanBills(ctx context.Context, query string, args ...any) ([]models.Bill, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query bills: %w", err)
	}
	defer rows.Close()

	items := make([]models.Bill, 0)
	for rows.Next() {
		var (
			bill      models.Bill
			endMonth  sql.NullString
			createdAt string
		)
		if err := rows.Scan(
			&bill.ID,
			&bill.Name,
			&bill.Amount,
			&bill.AmountMode,
			&bill.InterestRate,
			&bill.CategoryID,
			&bill.DueDay,
			&bill.Frequency,
			&bill.Recurrence,
			&bill.StartMonth,
			&endMonth,
			&bill.Notes,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan bill: %w", err)
		}
		if endMonth.Valid {
			value := endMonth.String
			bill.EndMonth = &value
		}
		parsed, err := parseDateTime(createdAt)
		if err != nil {
			return nil, err
		}
		bill.CreatedAt = parsed
		items = append(items, bill)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bills: %w", err)
	}

	for i := range items {
		members, err := s.listBillMemberIDs(ctx, items[i].ID)
		if err != nil {
			return nil, err
		}
		items[i].MemberIDs = members
	}
	return items, nil
}

func (s *Store) listBillMemberIDs(ctx context.Context, billID int) ([]int, error) {
	const query = `
		SELECT member_id
		FROM bill_members
		WHERE bill_id = ?
		ORDER BY member_id ASC
	`
	rows, err := s.db.QueryContext(ctx, query, billID)
	if err != nil {
		return nil, fmt.Errorf("list bill members: %w", err)
	}
	defer rows.Close()

	ids := make([]int, 0)
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("scan bill member: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate bill members: %w", err)
	}
	return ids, nil
}

func replaceBillMembersTx(ctx context.Context, tx *sql.Tx, billID int, memberIDs []int) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM bill_members WHERE bill_id = ?`, billID); err != nil {
		return fmt.Errorf("clear bill members: %w", err)
	}
	for _, memberID := range normalizeMemberIDs(memberIDs) {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO bill_members (bill_id, member_id) VALUES (?, ?)`,
			billID,
			memberID,
		); err != nil {
			return fmt.Errorf("insert bill member: %w", err)
		}
	}
	return nil
}

func normalizeMemberIDs(ids []int) []int {
	if len(ids) == 0 {
		return []int{}
	}
	seen := make(map[int]struct{}, len(ids))
	out := make([]int, 0, len(ids))
	for _, id := range ids {
		if id < 1 {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}

func nullString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}
