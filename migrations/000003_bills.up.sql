CREATE TABLE IF NOT EXISTS bills (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT    NOT NULL,
    amount      REAL    NOT NULL CHECK (amount > 0),
    category_id INTEGER NOT NULL,
    member_id   INTEGER,
    due_day     INTEGER NOT NULL DEFAULT 1 CHECK (due_day >= 1 AND due_day <= 31),
    -- ongoing = forever (internet, light); until = ends on end_month (subscription, credit)
    recurrence  TEXT    NOT NULL CHECK (recurrence IN ('ongoing', 'until')),
    start_month TEXT    NOT NULL,
    end_month   TEXT,
    notes       TEXT    NOT NULL DEFAULT '',
    created_at  TEXT    NOT NULL,
    FOREIGN KEY (category_id) REFERENCES categories(id) ON DELETE RESTRICT,
    FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE SET NULL,
    CHECK (
        (recurrence = 'ongoing' AND end_month IS NULL)
        OR (recurrence = 'until' AND end_month IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_bills_start_month ON bills (start_month);
CREATE INDEX IF NOT EXISTS idx_bills_end_month ON bills (end_month);
