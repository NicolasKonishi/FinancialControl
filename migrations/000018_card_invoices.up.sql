CREATE TABLE IF NOT EXISTS card_invoices (
    wallet_id             INTEGER NOT NULL,
    year                  INTEGER NOT NULL,
    month                 INTEGER NOT NULL CHECK (month >= 1 AND month <= 12),
    amount                REAL    NOT NULL DEFAULT 0 CHECK (amount >= 0),
    paid_amount           REAL    NOT NULL DEFAULT 0 CHECK (paid_amount >= 0),
    source                TEXT    NOT NULL DEFAULT 'calculated' CHECK (source IN ('calculated', 'statement')),
    statement_period_start TEXT,
    statement_period_end   TEXT,
    statement_balance      REAL,
    updated_at            TEXT    NOT NULL,
    PRIMARY KEY (wallet_id, year, month),
    FOREIGN KEY (wallet_id) REFERENCES wallets(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_card_invoices_due_month
    ON card_invoices (year, month);

-- This is a personal household app. Preserve the user's existing records while
-- turning the known Nubank card into an actual credit-card wallet.
INSERT INTO wallets (
    name, kind, member_id, balance, closing_day, due_day,
    credit_limit, invoice_balance, created_at
)
SELECT
    'Nubank', 'credit', m.id, 0, 14, 21,
    0, 0, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM members m
WHERE lower(m.name) = 'nicolas'
  AND NOT EXISTS (
      SELECT 1 FROM wallets w
      WHERE w.kind = 'credit'
        AND lower(w.name) LIKE '%nubank%'
        AND w.member_id = m.id
  );

UPDATE wallets
SET closing_day = 14, due_day = 21
WHERE kind = 'credit' AND lower(name) LIKE '%nubank%';

-- Bills previously entered under the "Cartão" category are components of the
-- Nubank invoice, not separate cash bills.
UPDATE bills
SET wallet_id = (
    SELECT w.id
    FROM wallets w
    JOIN members m ON m.id = w.member_id
    WHERE w.kind = 'credit'
      AND lower(w.name) LIKE '%nubank%'
      AND lower(m.name) = 'nicolas'
    ORDER BY w.id
    LIMIT 1
)
WHERE category_id IN (
    SELECT id FROM categories
    WHERE lower(name) IN ('cartão', 'cartao')
)
  AND wallet_id IS NULL
  AND EXISTS (
      SELECT 1
      FROM wallets w
      JOIN members m ON m.id = w.member_id
      WHERE w.kind = 'credit'
        AND lower(w.name) LIKE '%nubank%'
        AND lower(m.name) = 'nicolas'
  );

UPDATE categories
SET icon = 'shopping'
WHERE lower(name) IN ('cartão', 'cartao') AND icon IN ('salary', 'other');
