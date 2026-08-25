DROP INDEX IF EXISTS idx_bill_installments_due_month;
DROP INDEX IF EXISTS idx_bill_installments_bill_id;
DROP TABLE IF EXISTS bill_installments;
ALTER TABLE bills DROP COLUMN amount_mode;
