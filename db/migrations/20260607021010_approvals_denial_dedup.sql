-- +goose Up

ALTER TABLE approvals
    ADD COLUMN denial_reason text CHECK (char_length(denial_reason) <= 500),
    ADD COLUMN action_fingerprint text CHECK (char_length(action_fingerprint) <= 64);

CREATE INDEX idx_approvals_denial_dedup
    ON approvals (agent_id, approver_id, action_fingerprint, denied_at DESC)
    WHERE status = 'denied';

-- +goose Down

DROP INDEX IF EXISTS idx_approvals_denial_dedup;

ALTER TABLE approvals
    DROP COLUMN IF EXISTS action_fingerprint,
    DROP COLUMN IF EXISTS denial_reason;
