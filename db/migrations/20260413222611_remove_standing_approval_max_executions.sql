-- +goose Up

-- Standing approvals no longer track execution quotas; demote legacy rows.
UPDATE standing_approvals SET status = 'active' WHERE status = 'exhausted';

ALTER TABLE standing_approvals DROP CONSTRAINT IF EXISTS standing_approvals_status_check;
ALTER TABLE standing_approvals ADD CONSTRAINT standing_approvals_status_check
    CHECK (status IN ('active', 'expired', 'revoked'));

ALTER TABLE standing_approvals DROP COLUMN IF EXISTS max_executions;
ALTER TABLE standing_approvals DROP COLUMN IF EXISTS execution_count;
ALTER TABLE standing_approvals DROP COLUMN IF EXISTS exhausted_at;

-- +goose Down

ALTER TABLE standing_approvals
    ADD COLUMN max_executions int,
    ADD COLUMN execution_count int NOT NULL DEFAULT 0,
    ADD COLUMN exhausted_at timestamptz;

ALTER TABLE standing_approvals ADD CONSTRAINT standing_approvals_max_executions_positive
    CHECK (max_executions IS NULL OR max_executions > 0);
ALTER TABLE standing_approvals ADD CONSTRAINT standing_approvals_execution_count_nonneg
    CHECK (execution_count >= 0);

ALTER TABLE standing_approvals DROP CONSTRAINT IF EXISTS standing_approvals_status_check;
ALTER TABLE standing_approvals ADD CONSTRAINT standing_approvals_status_check
    CHECK (status IN ('active', 'expired', 'revoked', 'exhausted'));
