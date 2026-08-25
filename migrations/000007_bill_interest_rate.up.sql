ALTER TABLE bills ADD COLUMN interest_rate REAL NOT NULL DEFAULT 0;

UPDATE bills SET amount_mode = 'interest' WHERE amount_mode = 'schedule';
