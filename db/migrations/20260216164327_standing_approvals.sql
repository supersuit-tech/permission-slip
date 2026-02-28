-- +goose Up

CREATE TABLE standing_approvals (
    standing_approval_id text        PRIMARY KEY CHECK (char_length(standing_approval_id) <= 255),
    agent_id             text        NOT NULL REFERENCES agents(agent_id) ON DELETE CASCADE,
    user_id              uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    action_type          text        NOT NULL CHECK (char_length(action_type) <= 128),
    action_version       text        NOT NULL DEFAULT '1' CHECK (char_length(action_version) <= 10),
    constraints          jsonb       CHECK (pg_column_size(constraints) <= 65536),
    status               text        NOT NULL CHECK (status IN ('active', 'expired', 'revoked', 'exhausted')),
    max_executions       int         CHECK (max_executions > 0),
    execution_count      int         NOT NULL DEFAULT 0 CHECK (execution_count >= 0),
    starts_at            timestamptz NOT NULL,
    expires_at           timestamptz NOT NULL,
    created_at           timestamptz NOT NULL DEFAULT now(),
    revoked_at           timestamptz,
    expired_at           timestamptz,
    exhausted_at         timestamptz,
    CHECK (expires_at >= starts_at),
    CHECK (expires_at - starts_at <= interval '90 days')
);

CREATE INDEX idx_standing_approvals_agent_action_status ON standing_approvals (agent_id, action_type, status);

-- +goose Down
DROP TABLE IF EXISTS standing_approvals;
