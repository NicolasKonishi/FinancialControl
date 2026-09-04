package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/NicolasKonishi/FinancialControl/internal/models"
)

// FindStatementImportByHash returns a previous import of this file on this wallet.
func (s *Store) FindStatementImportByHash(ctx context.Context, walletID int, fileSHA256 string) (models.StatementImport, error) {
	const query = `
		SELECT id, wallet_id, statement_type, year, month, file_sha256, file_name, created_at
		FROM statement_imports
		WHERE wallet_id = ? AND file_sha256 = ?
	`
	return scanStatementImport(s.db.QueryRowContext(ctx, query, walletID, strings.ToLower(strings.TrimSpace(fileSHA256))))
}

// FindStatementImportByMonth returns a previous import of this competence on this wallet.
func (s *Store) FindStatementImportByMonth(ctx context.Context, walletID int, statementType string, year, month int) (models.StatementImport, error) {
	const query = `
		SELECT id, wallet_id, statement_type, year, month, file_sha256, file_name, created_at
		FROM statement_imports
		WHERE wallet_id = ? AND statement_type = ? AND year = ? AND month = ?
	`
	return scanStatementImport(s.db.QueryRowContext(ctx, query, walletID, statementType, year, month))
}

// CreateStatementImport records a successful statement upload so it cannot be imported again.
func (s *Store) CreateStatementImport(ctx context.Context, item models.StatementImport) (models.StatementImport, error) {
	const query = `
		INSERT INTO statement_imports (wallet_id, statement_type, year, month, file_sha256, file_name, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id, wallet_id, statement_type, year, month, file_sha256, file_name, created_at
	`
	now := formatDateTime(time.Now())
	return scanStatementImport(s.db.QueryRowContext(
		ctx,
		query,
		item.WalletID,
		item.StatementType,
		item.Year,
		item.Month,
		strings.ToLower(strings.TrimSpace(item.FileSHA256)),
		strings.TrimSpace(item.FileName),
		now,
	))
}

func scanStatementImport(row interface{ Scan(...any) error }) (models.StatementImport, error) {
	var (
		item      models.StatementImport
		createdAt string
	)
	err := row.Scan(
		&item.ID,
		&item.WalletID,
		&item.StatementType,
		&item.Year,
		&item.Month,
		&item.FileSHA256,
		&item.FileName,
		&createdAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return models.StatementImport{}, ErrNotFound
		}
		return models.StatementImport{}, fmt.Errorf("scan statement import: %w", err)
	}
	parsed, err := parseDateTime(createdAt)
	if err != nil {
		return models.StatementImport{}, err
	}
	item.CreatedAt = parsed
	return item, nil
}
