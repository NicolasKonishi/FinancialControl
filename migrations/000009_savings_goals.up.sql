CREATE TABLE IF NOT EXISTS savings_goals (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    name           TEXT    NOT NULL,
    target_amount  REAL    NOT NULL CHECK (target_amount > 0),
    monthly_amount REAL    NOT NULL CHECK (monthly_amount >= 0),
    notes          TEXT    NOT NULL DEFAULT '',
    created_at     TEXT    NOT NULL
);

CREATE TABLE IF NOT EXISTS savings_goal_members (
    goal_id   INTEGER NOT NULL,
    member_id INTEGER NOT NULL,
    PRIMARY KEY (goal_id, member_id),
    FOREIGN KEY (goal_id) REFERENCES savings_goals(id) ON DELETE CASCADE,
    FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS savings_month_amounts (
    goal_id    INTEGER NOT NULL,
    year       INTEGER NOT NULL,
    month      INTEGER NOT NULL CHECK (month >= 1 AND month <= 12),
    amount     REAL    NOT NULL CHECK (amount >= 0),
    saved_at   TEXT    NOT NULL,
    PRIMARY KEY (goal_id, year, month),
    FOREIGN KEY (goal_id) REFERENCES savings_goals(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_savings_month_amounts_month ON savings_month_amounts (year, month);
