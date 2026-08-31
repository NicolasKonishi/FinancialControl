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

// CreateTransaction inserts a transaction and moves wallet cash when a wallet is set.
func (s *Store) CreateTransaction(ctx context.Context, tx models.Transaction) (models.Transaction, error) {
	// Validate on s.db before BeginTx: the pool is MaxOpenConns(1), so a query
	// on s.db while a transaction holds the only connection deadlocks.
	if err := s.validateTxWallet(ctx, tx.MemberID, tx.WalletID); err != nil {
		return models.Transaction{}, err
	}

	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Transaction{}, fmt.Errorf("begin create transaction: %w", err)
	}
	defer dbTx.Rollback()

	const query = `
		INSERT INTO transactions (category_id, member_id, wallet_id, type, description, amount, date, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		RETURNING id, category_id, member_id, wallet_id, type, description, amount, date, created_at
	`

	now := formatDateTime(time.Now())
	created, err := scanTransaction(dbTx.QueryRowContext(
		ctx,
		query,
		tx.CategoryID,
		nullInt(tx.MemberID),
		nullInt(tx.WalletID),
		tx.Type,
		strings.TrimSpace(tx.Description),
		tx.Amount,
		formatDate(tx.Date),
		now,
	))
	if err != nil {
		return models.Transaction{}, fmt.Errorf("create transaction: %w", err)
	}

	if err := applyTransactionWallet(ctx, dbTx, nil, &created); err != nil {
		return models.Transaction{}, err
	}
	if err := dbTx.Commit(); err != nil {
		return models.Transaction{}, fmt.Errorf("commit create transaction: %w", err)
	}
	return created, nil
}

// ListTransactions returns all transactions ordered by date desc, then id desc.
func (s *Store) ListTransactions(ctx context.Context) ([]models.Transaction, error) {
	const query = `
		SELECT id, category_id, member_id, wallet_id, type, description, amount, date, created_at
		FROM transactions
		ORDER BY date DESC, id DESC
	`
	return s.scanTransactions(ctx, query)
}

// GetTransactionByID returns one transaction or ErrNotFound.
func (s *Store) GetTransactionByID(ctx context.Context, id int) (models.Transaction, error) {
	const query = `
		SELECT id, category_id, member_id, wallet_id, type, description, amount, date, created_at
		FROM transactions
		WHERE id = ?
	`

	tx, err := scanTransaction(s.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Transaction{}, ErrNotFound
	}
	if err != nil {
		return models.Transaction{}, fmt.Errorf("get transaction: %w", err)
	}
	return tx, nil
}

// UpdateTransaction replaces an existing transaction and moves wallet cash.
func (s *Store) UpdateTransaction(ctx context.Context, id int, tx models.Transaction) (models.Transaction, error) {
	existing, err := s.GetTransactionByID(ctx, id)
	if err != nil {
		return models.Transaction{}, err
	}
	if err := s.validateTxWallet(ctx, tx.MemberID, tx.WalletID); err != nil {
		return models.Transaction{}, err
	}

	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return models.Transaction{}, fmt.Errorf("begin update transaction: %w", err)
	}
	defer dbTx.Rollback()

	const query = `
		UPDATE transactions
		SET category_id = ?, member_id = ?, wallet_id = ?, type = ?, description = ?, amount = ?, date = ?
		WHERE id = ?
		RETURNING id, category_id, member_id, wallet_id, type, description, amount, date, created_at
	`

	updated, err := scanTransaction(dbTx.QueryRowContext(
		ctx,
		query,
		tx.CategoryID,
		nullInt(tx.MemberID),
		nullInt(tx.WalletID),
		tx.Type,
		strings.TrimSpace(tx.Description),
		tx.Amount,
		formatDate(tx.Date),
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Transaction{}, ErrNotFound
	}
	if err != nil {
		return models.Transaction{}, fmt.Errorf("update transaction: %w", err)
	}

	if err := applyTransactionWallet(ctx, dbTx, &existing, &updated); err != nil {
		return models.Transaction{}, err
	}
	if err := dbTx.Commit(); err != nil {
		return models.Transaction{}, fmt.Errorf("commit update transaction: %w", err)
	}
	return updated, nil
}

// DeleteTransaction removes a transaction and refunds its wallet movement.
func (s *Store) DeleteTransaction(ctx context.Context, id int) error {
	existing, err := s.GetTransactionByID(ctx, id)
	if err != nil {
		return err
	}

	dbTx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete transaction: %w", err)
	}
	defer dbTx.Rollback()

	if err := applyTransactionWallet(ctx, dbTx, &existing, nil); err != nil {
		return err
	}

	result, err := dbTx.ExecContext(ctx, `DELETE FROM transactions WHERE id = ?`, id)
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
	if err := dbTx.Commit(); err != nil {
		return fmt.Errorf("commit delete transaction: %w", err)
	}
	return nil
}

// ListTransactionsByMonth returns transactions whose date falls in year/month.
func (s *Store) ListTransactionsByMonth(ctx context.Context, year, month int) ([]models.Transaction, error) {
	start := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 1, 0)

	const query = `
		SELECT id, category_id, member_id, wallet_id, type, description, amount, date, created_at
		FROM transactions
		WHERE date >= ? AND date < ?
		ORDER BY date ASC, id ASC
	`
	return s.scanTransactions(ctx, query, formatDate(start), formatDate(end))
}

func (s *Store) validateTxWallet(ctx context.Context, memberID, walletID *int) error {
	if walletID == nil || *walletID < 1 {
		return nil
	}
	wallet, err := s.GetWalletByID(ctx, *walletID)
	if err != nil {
		return err
	}
	if memberID != nil && wallet.MemberID != nil && *wallet.MemberID != *memberID {
		if !models.IsCompanyWallet(wallet.Kind) {
			return ErrWalletOwner
		}
	}
	return nil
}

func applyTransactionWallet(ctx context.Context, tx *sql.Tx, previous, next *models.Transaction) error {
	if previous != nil && previous.WalletID != nil {
		if err := adjustWalletTx(ctx, tx, *previous.WalletID, -transactionWalletDelta(*previous)); err != nil {
			return err
		}
	}
	if next != nil && next.WalletID != nil {
		if err := adjustWalletTx(ctx, tx, *next.WalletID, transactionWalletDelta(*next)); err != nil {
			return err
		}
	}
	return nil
}

func transactionWalletDelta(tx models.Transaction) float64 {
	if tx.Type == models.TransactionTypeIncome {
		return tx.Amount
	}
	return -tx.Amount
}

func (s *Store) scanTransactions(ctx context.Context, query string, args ...any) ([]models.Transaction, error) {
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query transactions: %w", err)
	}
	defer rows.Close()

	items := make([]models.Transaction, 0)
	for rows.Next() {
		item, err := scanTransaction(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate transactions: %w", err)
	}
	return items, nil
}

func scanTransaction(row interface {
	Scan(dest ...any) error
}) (models.Transaction, error) {
	var (
		tx        models.Transaction
		memberID  sql.NullInt64
		walletID  sql.NullInt64
		date      string
		createdAt string
	)
	if err := row.Scan(
		&tx.ID,
		&tx.CategoryID,
		&memberID,
		&walletID,
		&tx.Type,
		&tx.Description,
		&tx.Amount,
		&date,
		&createdAt,
	); err != nil {
		return models.Transaction{}, err
	}
	tx.MemberID = fromNullInt(memberID)
	tx.WalletID = fromNullInt(walletID)
	return hydrateTransaction(tx, date, createdAt)
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
