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

// CreateTransaction inserts a transaction row.
func (s *Store) CreateTransaction(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
	const query = `
		INSERT INTO transactions (category_id, member_id, type, description, amount, date, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, category_id, member_id, type, description, amount, date, created_at
	`

	now := formatDateTime(time.Now())
	var (
		created   models.Transaction
		memberID  sql.NullInt64
		date      string
		createdAt string
	)
	err := s.db.QueryRowContext(
		ctx,
		query,
		tx.CategoryID,
		nullInt(tx.MemberID),
		tx.Type,
		strings.TrimSpace(tx.Description),
		tx.Amount,
		formatDate(tx.Date),
		now,
	).Scan(
		&created.ID,
		&created.CategoryID,
		&memberID,
		&created.Type,
		&created.Description,
		&created.Amount,
		&date,
		&createdAt,
	)
	if err != nil {
		return models.Transaction{}, fmt.Errorf("create transaction: %w", err)
	}
	created.MemberID = fromNullInt(memberID)
	return hydrateTransaction(created, date, createdAt)
}

// ListTransactions returns all transactions ordered by date desc, then id desc.
func (s *Store) ListTransactions(ctx context.Context) ([]models.Transaction, error) {
	const query = `
		SELECT id, category_id, member_id, type, description, amount, date, created_at
		FROM transactions
		ORDER BY date DESC, id DESC
	`
	return s.scanTransactions(ctx, query)
}

// GetTransactionByID returns one transaction or ErrNotFound.
func (s *Store) GetTransactionByID(ctx context.Context, id int) (models.Transaction, error) {
	const query = `
		SELECT id, category_id, member_id, type, description, amount, date, created_at
		FROM transactions
		WHERE id = ?
	`

	var (
		tx        models.Transaction
		memberID  sql.NullInt64
		date      string
		createdAt string
	)
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&tx.ID,
		&tx.CategoryID,
		&memberID,
		&tx.Type,
		&tx.Description,
		&tx.Amount,
		&date,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Transaction{}, ErrNotFound
	}
	if err != nil {
		return models.Transaction{}, fmt.Errorf("get transaction: %w", err)
	}
	tx.MemberID = fromNullInt(memberID)
	return hydrateTransaction(tx, date, createdAt)
}

// UpdateTransaction replaces an existing transaction.
func (s *Store) UpdateTransaction(ctx context.Context, id int, tx models.Transaction) (models.Transaction, error) {
	const query = `
		UPDATE transactions
		SET category_id = ?, member_id = ?, type = ?, description = ?, amount = ?, date = ?
		WHERE id = ?
		RETURNING id, category_id, member_id, type, description, amount, date, created_at
	`

	var (
		updated   models.Transaction
		memberID  sql.NullInt64
		date      string
		createdAt string
	)
	err := s.db.QueryRowContext(
		ctx,
		query,
		tx.CategoryID,
		nullInt(tx.MemberID),
		tx.Type,
		strings.TrimSpace(tx.Description),
		tx.Amount,
		formatDate(tx.Date),
		id,
	).Scan(
		&updated.ID,
		&updated.CategoryID,
		&memberID,
		&updated.Type,
		&updated.Description,
		&updated.Amount,
		&date,
		&createdAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.Transaction{}, ErrNotFound
	}
	if err != nil {
		return models.Transaction{}, fmt.Errorf("update transaction: %w", err)
	}
	updated.MemberID = fromNullInt(memberID)
	return hydrateTransaction(updated, date, createdAt)
}

// DeleteTransaction removes a transaction by id.
func (s *Store) DeleteTransaction(ctx context.Context, id int) error {
	const query = `DELETE FROM transactions WHERE id = ?`

	result, err := s.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("delete transaction: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete transaction rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

// ListTransactionsByMonth returns transactions whose date falls in year/month.
func (s *Store) ListTransactionsByMonth(ctx context.Context, year, month int) ([]models.Transaction, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	const query = `
		SELECT id, category_id, member_id, type, description, amount, date, created_at
		FROM transactions
		WHERE date >= ? AND date < ?
		ORDER BY date ASC, id ASC
	`
	return s.scanTransactions(ctx, query, formatDate(start), formatDate(end))
}

func (s *Store) scanTransactions(ctx context.Context, query string, args ...any) ([]models.Transaction, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	items := make([]models.Transaction, 0)
	for rows.Next() {
		var (
			tx        models.Transaction
			memberID  sql.NullInt64
			date      string
			createdAt string
		)
		if err := rows.Scan(
			&tx.ID,
			&tx.CategoryID,
			&memberID,
			&tx.Type,
			&tx.Description,
			&tx.Amount,
			&date,
			&createdAt,
		); err != nil {
			return nil, fmt.Errorf("scan transaction: %w", err)
		}
		tx.MemberID = fromNullInt(memberID)
		hydrated, err := hydrateTransaction(tx, date, createdAt)
		if err != nil {
			return nil, err
		}
		items = append(items, hydrated)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}
	return items, nil
}

func hydrateTransaction(tx models.Transaction, date, createdAt string) (models.Transaction, error) {
	parsedDate, err := parseDate(date)
	if err != nil {
		return models.Transaction{}, err
	}
	parsedCreatedAt, err := parseDateTime(createdAt)
	if err != nil {
		return models.Transaction{}, err
	}
	tx.Date = parsedDate
	tx.CreatedAt = parsedCreatedAt
	return tx, nil
}

func nullInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func fromNullInt(value sql.NullInt64) *int {
	if !value.Valid {
		return nil
	}
	id := int(value.Int64)
	return &id
}
