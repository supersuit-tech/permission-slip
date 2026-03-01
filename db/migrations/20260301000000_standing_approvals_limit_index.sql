-- +goose Up
-- Partial index to support fast plan-limit COUNT queries on active standing
-- approvals per user. Only includes status='active' rows since those are the
-- only ones counted toward plan limits.
CREATE INDEX IF NOT EXISTS idx_standing_approvals_user_active
    ON standing_approvals (user_id)
    WHERE status = 'active';

-- +goose Down
DROP INDEX IF EXISTS idx_standing_approvals_user_active;
