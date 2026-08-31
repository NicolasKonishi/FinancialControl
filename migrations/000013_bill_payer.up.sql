ALTER TABLE bill_payments ADD COLUMN paid_by_member_id INTEGER REFERENCES members(id);
ALTER TABLE bill_payments ADD COLUMN wallet_id INTEGER REFERENCES wallets(id);
ALTER TABLE bill_payments ADD COLUMN amount REAL NOT NULL DEFAULT 0;

INSERT INTO members (name, monthly_salary, created_at)
SELECT 'empresa', 0, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM members WHERE lower(name) = 'empresa');

UPDATE wallets
SET member_id = (SELECT id FROM members WHERE lower(name) = 'empresa')
WHERE name = 'Conta da empresa'
  AND member_id IN (SELECT id FROM members WHERE lower(name) = 'nicolas');
