DROP INDEX IF EXISTS idx_card_invoices_due_month;
DROP TABLE IF EXISTS card_invoices;

UPDATE bills
SET wallet_id = NULL
WHERE category_id IN (
    SELECT id FROM categories
    WHERE lower(name) IN ('cartão', 'cartao')
)
  AND wallet_id IN (
    SELECT id FROM wallets
    WHERE kind = 'credit' AND lower(name) LIKE '%nubank%'
  );
