ALTER TABLE bills ADD COLUMN amount_mode TEXT NOT NULL DEFAULT 'fixed';

CREATE TABLE IF NOT EXISTS bill_installments (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    bill_id   INTEGER NOT NULL,
    due_month TEXT    NOT NULL,
    amount    REAL    NOT NULL CHECK (amount > 0),
    notes     TEXT    NOT NULL DEFAULT '',
    FOREIGN KEY (bill_id) REFERENCES bills(id) ON DELETE CASCADE,
    UNIQUE (bill_id, due_month)
);

CREATE INDEX IF NOT EXISTS idx_bill_installments_bill_id ON bill_installments (bill_id);
CREATE INDEX IF NOT EXISTS idx_bill_installments_due_month ON bill_installments (due_month);
