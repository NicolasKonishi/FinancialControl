INSERT INTO members (name, monthly_salary, created_at)
SELECT 'empresa', 0, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM members WHERE lower(name) = 'empresa');

UPDATE wallets
SET member_id = (SELECT id FROM members WHERE lower(name) = 'empresa')
WHERE kind = 'company'
  AND member_id IN (SELECT id FROM members WHERE lower(name) = 'nicolas')
  AND EXISTS (SELECT 1 FROM members WHERE lower(name) = 'empresa');

UPDATE bills
SET wallet_id = NULL
WHERE wallet_id IN (
  SELECT id FROM wallets
  WHERE kind = 'company'
    AND member_id IN (SELECT id FROM members WHERE lower(name) = 'empresa')
);
