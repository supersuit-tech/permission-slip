-- +goose Up
-- Add index to support ListStandingApprovalsByAgent query which filters by
-- (agent_id, status) and orders by (created_at DESC, standing_approval_id DESC).
-- The existing idx_standing_approvals_agent_action_status index includes
-- action_type as a middle column, so it can't efficiently serve this query.
CREATE INDEX IF NOT EXISTS idx_standing_approvals_agent_status_created
    ON standing_approvals (agent_id, status, created_at DESC, standing_approval_id DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_standing_approvals_agent_status_created;
