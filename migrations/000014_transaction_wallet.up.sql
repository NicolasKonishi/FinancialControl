ALTER TABLE transactions ADD COLUMN wallet_id INTEGER REFERENCES wallets(id);
CREATE INDEX IF NOT EXISTS idx_transactions_wallet_id ON transactions (wallet_id);
