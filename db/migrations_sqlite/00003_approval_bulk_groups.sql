-- +goose Up

CREATE TABLE approval_bulk_groups (
    bulk_group_id   TEXT PRIMARY KEY,
    agent_id        INTEGER NOT NULL,
    approver_id     TEXT NOT NULL,
    action_type     TEXT NOT NULL,
    item_count      INTEGER NOT NULL CHECK (item_count > 0),
    allow_mixed     INTEGER NOT NULL DEFAULT 0 CHECK (allow_mixed IN (0, 1)),
    expires_at      TEXT NOT NULL,
    created_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (approver_id) REFERENCES profiles(id) ON DELETE CASCADE,
    CONSTRAINT approval_bulk_groups_bulk_group_id_check CHECK (length(bulk_group_id) <= 255),
    CONSTRAINT approval_bulk_groups_action_type_check CHECK (length(action_type) <= 128)
);

CREATE INDEX idx_approval_bulk_groups_approver_created
    ON approval_bulk_groups (approver_id, created_at DESC, bulk_group_id DESC);

CREATE INDEX idx_approval_bulk_groups_agent_created
    ON approval_bulk_groups (agent_id, created_at DESC);

ALTER TABLE approvals ADD COLUMN bulk_group_id TEXT REFERENCES approval_bulk_groups(bulk_group_id) ON DELETE SET NULL;

CREATE INDEX idx_approvals_bulk_group_id ON approvals (bulk_group_id) WHERE bulk_group_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_approvals_bulk_group_id;
-- SQLite does not support DROP COLUMN in older versions; recreate would be needed for full down.
-- For tests, dropping the table is sufficient.
DROP TABLE IF EXISTS approval_bulk_groups;
