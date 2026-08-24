CREATE TABLE IF NOT EXISTS bill_members (
    bill_id   INTEGER NOT NULL,
    member_id INTEGER NOT NULL,
    PRIMARY KEY (bill_id, member_id),
    FOREIGN KEY (bill_id) REFERENCES bills(id) ON DELETE CASCADE,
    FOREIGN KEY (member_id) REFERENCES members(id) ON DELETE CASCADE
);

-- Preserve existing single-member links, if any.
INSERT OR IGNORE INTO bill_members (bill_id, member_id)
SELECT id, member_id
FROM bills
WHERE member_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_bill_members_member_id ON bill_members (member_id);
