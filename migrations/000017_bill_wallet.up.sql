ALTER TABLE bills ADD COLUMN wallet_id INTEGER REFERENCES wallets(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_bills_wallet_id ON bills (wallet_id);
