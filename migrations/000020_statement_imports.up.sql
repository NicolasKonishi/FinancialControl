CREATE TABLE IF NOT EXISTS statement_imports (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    wallet_id       INTEGER NOT NULL,
    statement_type  TEXT    NOT NULL CHECK (statement_type IN ('credit_card', 'account')),
    year            INTEGER NOT NULL,
    month           INTEGER NOT NULL CHECK (month >= 1 AND month <= 12),
    file_sha256     TEXT    NOT NULL,
    file_name       TEXT    NOT NULL DEFAULT '',
    created_at      TEXT    NOT NULL,
    FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE,
    UNIQUE (wallet_id, statement_type, year, month),
    UNIQUE (wallet_id, file_sha256)
);

CREATE INDEX IF NOT EXISTS idx_statement_imports_month
    ON statement_imports (wallet_id, statement_type, year, month);
