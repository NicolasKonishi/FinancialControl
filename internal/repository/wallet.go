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

// CreateWallet inserts an account, box, or benefit balance.
func (s *Store) CreateWallet(ctx context.Context, input models.CreateWalletInput) (models.Wallet, error) {
	const query = `
		INSERT INTO wallets (name, kind, member_id, balance, created_at)
		VALUES (?, ?, ?, ?, ?)
		RETURNING id, name, kind, member_id, balance, created_at
	`

	now := formatDateTime(time.Now())
	wallet, err := scanWallet(s.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(input.Name),
		models.NormalizeWalletKind(input.Kind),
		nullInt(input.MemberID),
		input.Balance,
		now,
	))
	if err != nil {
		return models.Wallet{}, fmt.Errorf("create wallet: %w", err)
	}
	return wallet, nil
}

// ListWallets returns all wallets.
func (s *Store) ListWallets(ctx context.Context) ([]models.Wallet, error) {
	const query = `
		SELECT id, name, kind, member_id, balance, created_at
		FROM wallets
		ORDER BY CASE WHEN member_id IS NULL THEN 0 ELSE 1 END, member_id, id
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list wallets: %w", err)
	}
	defer rows.Close()

	items := make([]models.Wallet, 0)
	for rows.Next() {
		wallet, err := scanWallet(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, wallet)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate wallets: %w", err)
	}
	return items, nil
}

// GetWalletByID returns one wallet or ErrNotFound.
func (s *Store) GetWalletByID(ctx context.Context, id int) (models.Wallet, error) {
	const query = `
		SELECT id, name, kind, member_id, balance, created_at
		FROM wallets
		WHERE id = ?
	`

	wallet, err := scanWallet(s.db.QueryRowContext(ctx, query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Wallet{}, ErrNotFound
	}
	if err != nil {
		return models.Wallet{}, fmt.Errorf("get wallet: %w", err)
	}
	return wallet, nil
}

// UpdateWallet updates a wallet.
func (s *Store) UpdateWallet(ctx context.Context, id int, input models.UpdateWalletInput) (models.Wallet, error) {
	const query = `
		UPDATE wallets
		SET name = ?, kind = ?, member_id = ?, balance = ?
		WHERE id = ?
		RETURNING id, name, kind, member_id, balance, created_at
	`

	wallet, err := scanWallet(s.db.QueryRowContext(
		ctx,
		query,
		strings.TrimSpace(input.Name),
		models.NormalizeWalletKind(input.Kind),
		nullInt(input.MemberID),
		input.Balance,
		id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return models.Wallet{}, ErrNotFound
	}
	if err != nil {
		return models.Wallet{}, fmt.Errorf("update wallet: %w", err)
	}
	return wallet, nil
}

// DeleteWallet removes a wallet.
func (s *Store) DeleteWallet(ctx context.Context, id int) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM wallets WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete wallet: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete wallet rows affected: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func scanWallet(row interface {
	Scan(dest ...any) error
}) (models.Wallet, error) {
	var (
		wallet    models.Wallet
		memberID  sql.NullInt64
		createdAt string
	)
	if err := row.Scan(&wallet.ID, &wallet.Name, &wallet.Kind, &memberID, &wallet.Balance, &createdAt); err != nil {
		return models.Wallet{}, err
	}
	wallet.MemberID = fromNullInt(memberID)
	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.Wallet{}, err
	}
	wallet.CreatedAt = parsed
	return wallet, nil
}
