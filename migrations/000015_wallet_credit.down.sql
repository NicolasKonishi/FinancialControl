PRAGMA foreign_keys = OFF;

CREATE TABLE wallets_new (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    kind       TEXT    NOT NULL CHECK (kind IN ('checking', 'savings', 'benefit', 'company')),
    member_id  INTEGER REFERENCES members(id) ON DELETE CASCADE,
    balance    REAL    NOT NULL DEFAULT 0,
    created_at TEXT    NOT NULL
);

INSERT INTO wallets_new (id, name, kind, member_id, balance, created_at)
SELECT id, name, CASE WHEN kind = 'credit' THEN 'benefit' ELSE kind END, member_id, balance, created_at
FROM wallets;

DROP TABLE wallets;
ALTER TABLE wallets_new RENAME TO wallets;

CREATE INDEX IF NOT EXISTS idx_wallets_member_id ON wallets (member_id);

PRAGMA foreign_keys = ON;
