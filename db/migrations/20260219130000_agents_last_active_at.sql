-- +goose Up
ALTER TABLE agents ADD COLUMN last_active_at timestamptz;

-- Covers the correlated COUNT subquery in GET /agents:
--   WHERE approvals.agent_id = ? AND approvals.created_at > now() - interval '30 days'
CREATE INDEX idx_approvals_agent_created ON approvals (agent_id, created_at);

-- +goose Down
DROP INDEX IF EXISTS idx_approvals_agent_created;
ALTER TABLE agents DROP COLUMN IF EXISTS last_active_at;
