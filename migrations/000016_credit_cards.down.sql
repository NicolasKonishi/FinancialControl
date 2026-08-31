UPDATE wallets
SET balance = CASE WHEN kind = 'credit' THEN -invoice_balance ELSE balance END
WHERE kind = 'credit';

ALTER TABLE wallets DROP COLUMN closing_day;
ALTER TABLE wallets DROP COLUMN due_day;
ALTER TABLE wallets DROP COLUMN credit_limit;
ALTER TABLE wallets DROP COLUMN invoice_balance;
