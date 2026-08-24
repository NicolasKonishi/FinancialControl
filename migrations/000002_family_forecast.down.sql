-- Best-effort down migration for SQLite (cannot drop columns easily).
DROP INDEX IF EXISTS idx_transactions_member_id;
DROP TABLE IF EXISTS members;
