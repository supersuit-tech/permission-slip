-- +goose Up
-- Covers the GET /agents pagination query:
--   WHERE approver_id = $1 ORDER BY created_at DESC, agent_id DESC
CREATE INDEX idx_agents_approver_created ON agents (approver_id, created_at DESC, agent_id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_agents_approver_created;
