UPDATE wallets
SET member_id = (SELECT id FROM members WHERE lower(name) = 'nicolas')
WHERE name = 'Conta da empresa'
  AND member_id IN (SELECT id FROM members WHERE lower(name) = 'empresa');

DELETE FROM members WHERE lower(name) = 'empresa';

ALTER TABLE bill_payments DROP COLUMN amount;
ALTER TABLE bill_payments DROP COLUMN wallet_id;
ALTER TABLE bill_payments DROP COLUMN paid_by_member_id;
