ALTER TABLE bills ADD COLUMN source TEXT NOT NULL DEFAULT 'manual'
    CHECK (source IN ('manual', 'statement'));
ALTER TABLE bills ADD COLUMN installment_start INTEGER NOT NULL DEFAULT 0
    CHECK (installment_start >= 0);
ALTER TABLE bills ADD COLUMN installment_total INTEGER NOT NULL DEFAULT 0
    CHECK (installment_total >= 0 AND installment_total <= 120);

