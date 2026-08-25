-- How often the bill charges. Existing rows default to monthly.
ALTER TABLE bills ADD COLUMN frequency TEXT NOT NULL DEFAULT 'monthly';
