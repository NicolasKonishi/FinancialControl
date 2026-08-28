CREATE TABLE IF NOT EXISTS bill_payments (
    bill_id INTEGER NOT NULL,
    year    INTEGER NOT NULL,
    month   INTEGER NOT NULL CHECK (month >= 1 AND month <= 12),
    paid_at TEXT    NOT NULL,
    PRIMARY KEY (bill_id, year, month),
    FOREIGN KEY (bill_id) REFERENCES bills(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_bill_payments_month ON bill_payments (year, month);
