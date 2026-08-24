package database

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite" // Pure-Go SQLite driver for database/sql (no CGO).
)

// Connect opens an SQLite database file and enables foreign keys.
func Connect(ctx context.Context, sqlitePath string) (*sql.DB, error) {
	dir := filepath.Dir(sqlitePath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("create sqlite directory: %w", err)
		}
	}

	// modernc.org/sqlite registers the driver name "sqlite".
	db, err := sql.Open("sqlite", sqlitePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}

	// SQLite handles concurrent writes poorly; keep a small pool.
	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}

	if _, err := db.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	return db, nil
}
