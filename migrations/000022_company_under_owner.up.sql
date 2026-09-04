-- Company cash is a personal sub-account, not a fake family member.
-- Prefer nicolas as owner (how the wallet was created); otherwise first real person.

UPDATE wallets
SET member_id = (
  SELECT id FROM members
  WHERE lower(name) <> 'empresa'
  ORDER BY CASE WHEN lower(name) = 'nicolas' THEN 0 ELSE 1 END, id
  LIMIT 1
)
WHERE member_id IN (SELECT id FROM members WHERE lower(name) = 'empresa')
  AND EXISTS (SELECT 1 FROM members WHERE lower(name) <> 'empresa');

UPDATE bills
SET wallet_id = (
  SELECT w.id FROM wallets w
  WHERE w.kind = 'company'
  ORDER BY CASE
    WHEN w.member_id = (SELECT id FROM members WHERE lower(name) = 'nicolas' LIMIT 1) THEN 0
    ELSE 1
  END, w.id
  LIMIT 1
)
WHERE (wallet_id IS NULL OR wallet_id = 0)
  AND id IN (
    SELECT bill_id FROM bill_members
    WHERE member_id IN (SELECT id FROM members WHERE lower(name) = 'empresa')
  )
  AND EXISTS (SELECT 1 FROM wallets WHERE kind = 'company');

INSERT OR IGNORE INTO bill_members (bill_id, member_id)
SELECT bm.bill_id, (
  SELECT id FROM members
  WHERE lower(name) <> 'empresa'
  ORDER BY CASE WHEN lower(name) = 'nicolas' THEN 0 ELSE 1 END, id
  LIMIT 1
)
FROM bill_members bm
WHERE bm.member_id IN (SELECT id FROM members WHERE lower(name) = 'empresa')
  AND EXISTS (SELECT 1 FROM members WHERE lower(name) <> 'empresa');

DELETE FROM bill_members
WHERE member_id IN (SELECT id FROM members WHERE lower(name) = 'empresa');

INSERT OR IGNORE INTO savings_goal_members (goal_id, member_id)
SELECT sgm.goal_id, (
  SELECT id FROM members
  WHERE lower(name) <> 'empresa'
  ORDER BY CASE WHEN lower(name) = 'nicolas' THEN 0 ELSE 1 END, id
  LIMIT 1
)
FROM savings_goal_members sgm
WHERE sgm.member_id IN (SELECT id FROM members WHERE lower(name) = 'empresa')
  AND EXISTS (SELECT 1 FROM members WHERE lower(name) <> 'empresa');

DELETE FROM savings_goal_members
WHERE member_id IN (SELECT id FROM members WHERE lower(name) = 'empresa');

UPDATE transactions
SET member_id = (
  SELECT id FROM members
  WHERE lower(name) <> 'empresa'
  ORDER BY CASE WHEN lower(name) = 'nicolas' THEN 0 ELSE 1 END, id
  LIMIT 1
)
WHERE member_id IN (SELECT id FROM members WHERE lower(name) = 'empresa')
  AND EXISTS (SELECT 1 FROM members WHERE lower(name) <> 'empresa');

UPDATE bill_payments
SET paid_by_member_id = (
  SELECT id FROM members
  WHERE lower(name) <> 'empresa'
  ORDER BY CASE WHEN lower(name) = 'nicolas' THEN 0 ELSE 1 END, id
  LIMIT 1
)
WHERE paid_by_member_id IN (SELECT id FROM members WHERE lower(name) = 'empresa')
  AND EXISTS (SELECT 1 FROM members WHERE lower(name) <> 'empresa');

DELETE FROM member_save_targets
WHERE member_id IN (SELECT id FROM members WHERE lower(name) = 'empresa');

DELETE FROM members
WHERE lower(name) = 'empresa';
