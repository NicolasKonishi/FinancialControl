PRAGMA foreign_keys = OFF;

CREATE TABLE savings_goals_new (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    name           TEXT    NOT NULL,
    target_amount  REAL    NOT NULL CHECK (target_amount >= 0),
    monthly_amount REAL    NOT NULL CHECK (monthly_amount >= 0),
    notes          TEXT    NOT NULL DEFAULT '',
    end_kind       TEXT    NOT NULL DEFAULT 'amount' CHECK (end_kind IN ('none', 'date', 'amount')),
    end_month      TEXT,
    cdi_annual     REAL    NOT NULL DEFAULT 0,
    created_at     TEXT    NOT NULL
);

INSERT INTO savings_goals_new (id, name, target_amount, monthly_amount, notes, end_kind, created_at)
SELECT id, name, target_amount, monthly_amount, notes, 'amount', created_at
FROM savings_goals;

DROP TABLE savings_goals;
ALTER TABLE savings_goals_new RENAME TO savings_goals;

PRAGMA foreign_keys = ON;
