-- +goose Up
-- Bulk approval groups: N same-type actions submitted as one notification/review unit.

CREATE TABLE approval_bulk_groups (
    bulk_group_id   text        PRIMARY KEY CHECK (char_length(bulk_group_id) <= 255),
    agent_id        bigint      NOT NULL REFERENCES agents(agent_id) ON DELETE CASCADE,
    approver_id     uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    action_type     text        NOT NULL CHECK (char_length(action_type) <= 128),
    item_count      int         NOT NULL CHECK (item_count > 0),
    allow_mixed     boolean     NOT NULL DEFAULT false,
    expires_at      timestamptz NOT NULL,
    created_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_approval_bulk_groups_approver_created
    ON approval_bulk_groups (approver_id, created_at DESC, bulk_group_id DESC);

CREATE INDEX idx_approval_bulk_groups_agent_created
    ON approval_bulk_groups (agent_id, created_at DESC);

ALTER TABLE approvals ADD COLUMN bulk_group_id text REFERENCES approval_bulk_groups(bulk_group_id) ON DELETE SET NULL;

CREATE INDEX idx_approvals_bulk_group_id ON approvals (bulk_group_id) WHERE bulk_group_id IS NOT NULL;

ALTER TABLE approval_bulk_groups ENABLE ROW LEVEL SECURITY;
CREATE POLICY app_backend_all ON approval_bulk_groups FOR ALL TO app_backend USING (true) WITH CHECK (true);

-- +goose Down
DROP INDEX IF EXISTS idx_approvals_bulk_group_id;
ALTER TABLE approvals DROP COLUMN IF EXISTS bulk_group_id;
DROP TABLE IF EXISTS approval_bulk_groups;
