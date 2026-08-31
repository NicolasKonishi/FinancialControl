DELETE FROM savings_goals WHERE name IN ('Caixinha conjunta', 'Caixinha');

INSERT INTO wallets (name, kind, member_id, balance, created_at)
SELECT 'Caixinha', 'savings', m.id, 2002.48, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM members m
WHERE lower(m.name) = 'nicolas'
  AND NOT EXISTS (SELECT 1 FROM wallets WHERE name = 'Caixinha' AND member_id = m.id);

INSERT INTO wallets (name, kind, member_id, balance, created_at)
SELECT 'Caixinha conjunta', 'savings', NULL, 1603.60, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM wallets WHERE name = 'Caixinha conjunta' AND member_id IS NULL);

ALTER TABLE savings_goals DROP COLUMN opening_amount;
