-- +goose Up
-- Covers the GET /approvals cursor-based pagination query:
--   WHERE approver_id = $1 ... ORDER BY created_at DESC, approval_id DESC
-- Mirrors idx_agents_approver_created on the agents table.
CREATE INDEX idx_approvals_approver_created ON approvals (approver_id, created_at DESC, approval_id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_approvals_approver_created;
