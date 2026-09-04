-- How much each person plans to set aside in a given month.
CREATE TABLE IF NOT EXISTS member_save_targets (
    member_id INTEGER NOT NULL,
    year      INTEGER NOT NULL,
    month     INTEGER NOT NULL CHECK (month >= 1 AND month <= 12),
    amount    REAL    NOT NULL CHECK (amount >= 0),
    PRIMARY KEY (member_id, year, month),
    FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_member_save_targets_month
    ON member_save_targets (year, month);
