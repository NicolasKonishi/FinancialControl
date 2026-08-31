CREATE TABLE IF NOT EXISTS wallets (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL,
    kind       TEXT    NOT NULL CHECK (kind IN ('checking', 'savings', 'benefit', 'company')),
    member_id  INTEGER REFERENCES members(id) ON DELETE CASCADE,
    balance    REAL    NOT NULL DEFAULT 0,
    created_at TEXT    NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_wallets_member_id ON wallets (member_id);

-- Snapshot of 28 Aug 2026. Idempotent per wallet name + owner.
INSERT INTO wallets (name, kind, member_id, balance, created_at)
SELECT 'Conta', 'checking', m.id, 1186.72, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM members m
WHERE lower(m.name) = 'nicolas'
  AND NOT EXISTS (SELECT 1 FROM wallets WHERE name = 'Conta' AND member_id = m.id);

INSERT INTO wallets (name, kind, member_id, balance, created_at)
SELECT 'Caixinha', 'savings', m.id, 2002.48, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM members m
WHERE lower(m.name) = 'nicolas'
  AND NOT EXISTS (SELECT 1 FROM wallets WHERE name = 'Caixinha' AND member_id = m.id);

INSERT INTO wallets (name, kind, member_id, balance, created_at)
SELECT 'Conta da empresa', 'company', m.id, 3042, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM members m
WHERE lower(m.name) = 'nicolas'
  AND NOT EXISTS (SELECT 1 FROM wallets WHERE name = 'Conta da empresa' AND member_id = m.id);

INSERT INTO wallets (name, kind, member_id, balance, created_at)
SELECT 'Caixinha conjunta', 'savings', NULL, 1603.60, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM wallets WHERE name = 'Caixinha conjunta' AND member_id IS NULL);

INSERT INTO wallets (name, kind, member_id, balance, created_at)
SELECT 'Conta', 'checking', m.id, 464.99, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM members m
WHERE lower(m.name) = 'michele'
  AND NOT EXISTS (SELECT 1 FROM wallets WHERE name = 'Conta' AND member_id = m.id);

INSERT INTO wallets (name, kind, member_id, balance, created_at)
SELECT 'Refeição', 'benefit', m.id, 623.12, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM members m
WHERE lower(m.name) = 'michele'
  AND NOT EXISTS (SELECT 1 FROM wallets WHERE name = 'Refeição' AND member_id = m.id);

INSERT INTO wallets (name, kind, member_id, balance, created_at)
SELECT 'Mobilidade', 'benefit', m.id, 546.90, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM members m
WHERE lower(m.name) = 'michele'
  AND NOT EXISTS (SELECT 1 FROM wallets WHERE name = 'Mobilidade' AND member_id = m.id);

INSERT INTO wallets (name, kind, member_id, balance, created_at)
SELECT 'Alimentação', 'benefit', m.id, 54.77, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
FROM members m
WHERE lower(m.name) = 'michele'
  AND NOT EXISTS (SELECT 1 FROM wallets WHERE name = 'Alimentação' AND member_id = m.id);
