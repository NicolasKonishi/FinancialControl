ALTER TABLE wallets ADD COLUMN closing_day INTEGER;
ALTER TABLE wallets ADD COLUMN due_day INTEGER;
ALTER TABLE wallets ADD COLUMN credit_limit REAL NOT NULL DEFAULT 0;
ALTER TABLE wallets ADD COLUMN invoice_balance REAL NOT NULL DEFAULT 0;

UPDATE wallets
SET
    invoice_balance = CASE WHEN kind = 'credit' AND balance < 0 THEN -balance ELSE 0 END,
    balance = CASE WHEN kind = 'credit' THEN 0 ELSE balance END
WHERE kind = 'credit';
