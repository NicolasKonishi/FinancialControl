ALTER TABLE savings_goals ADD COLUMN opening_amount REAL NOT NULL DEFAULT 0;

DELETE FROM wallets
WHERE name = 'Caixinha conjunta' AND member_id IS NULL;

DELETE FROM wallets
WHERE name = 'Caixinha'
  AND member_id IN (SELECT id FROM members WHERE lower(name) = 'nicolas');

INSERT INTO savings_goals (name, target_amount, monthly_amount, notes, end_kind, end_month, cdi_annual, opening_amount, created_at)
SELECT 'Caixinha conjunta', 0, 0, '', 'none', NULL, 14.15, 1603.60, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM savings_goals WHERE name = 'Caixinha conjunta');

INSERT INTO savings_goal_members (goal_id, member_id)
SELECT g.id, m.id
FROM savings_goals g
CROSS JOIN members m
WHERE g.name = 'Caixinha conjunta'
  AND NOT EXISTS (
    SELECT 1 FROM savings_goal_members sgm
    WHERE sgm.goal_id = g.id AND sgm.member_id = m.id
  );

INSERT INTO savings_goals (name, target_amount, monthly_amount, notes, end_kind, end_month, cdi_annual, opening_amount, created_at)
SELECT 'Caixinha', 0, 0, '', 'none', NULL, 14.15, 2002.48, strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
WHERE NOT EXISTS (SELECT 1 FROM savings_goals WHERE name = 'Caixinha');

INSERT INTO savings_goal_members (goal_id, member_id)
SELECT g.id, m.id
FROM savings_goals g
JOIN members m ON lower(m.name) = 'nicolas'
WHERE g.name = 'Caixinha'
  AND NOT EXISTS (
    SELECT 1 FROM savings_goal_members sgm
    WHERE sgm.goal_id = g.id AND sgm.member_id = m.id
  );
