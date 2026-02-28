-- +goose Up

CREATE TABLE approvals (
    approval_id            text        PRIMARY KEY CHECK (char_length(approval_id) <= 255),
    agent_id               text        NOT NULL REFERENCES agents(agent_id) ON DELETE CASCADE,
    approver_id            uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    action                 jsonb       NOT NULL CHECK (pg_column_size(action) <= 65536),
    context                jsonb       NOT NULL CHECK (pg_column_size(context) <= 65536),
    status                 text        NOT NULL CHECK (status IN ('pending', 'approved', 'denied', 'cancelled')),
    confirmation_code_hash text        CHECK (char_length(confirmation_code_hash) <= 128),
    verification_attempts  int         NOT NULL DEFAULT 0 CHECK (verification_attempts >= 0),
    token_jti              text        UNIQUE CHECK (char_length(token_jti) <= 255),
    expires_at             timestamptz NOT NULL,
    approved_at            timestamptz,
    denied_at              timestamptz,
    cancelled_at           timestamptz,
    created_at             timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_approvals_agent_status ON approvals (agent_id, status);
CREATE INDEX idx_approvals_expires_at ON approvals (expires_at);

CREATE TABLE request_ids (
    request_id text        PRIMARY KEY CHECK (char_length(request_id) <= 255),
    agent_id   text        NOT NULL REFERENCES agents(agent_id) ON DELETE CASCADE,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_request_ids_created_at ON request_ids (created_at);

CREATE TABLE consumed_tokens (
    jti         text        PRIMARY KEY CHECK (char_length(jti) <= 255),
    consumed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_consumed_tokens_consumed_at ON consumed_tokens (consumed_at);

-- pg_cron jobs for TTL cleanup.
-- Uses try_cron_schedule() from the profiles migration to gracefully skip
-- when pg_cron is not available (e.g. local development).

SELECT try_cron_schedule(
    'cleanup_request_ids',
    '*/5 * * * *',
    $$DELETE FROM request_ids WHERE created_at < now() - interval '300 seconds'$$
);

SELECT try_cron_schedule(
    'cleanup_consumed_tokens',
    '0 * * * *',
    $$DELETE FROM consumed_tokens WHERE consumed_at < now() - interval '24 hours'$$
);

-- +goose Down

-- Unschedule pg_cron jobs before dropping tables.
SELECT try_cron_unschedule('cleanup_request_ids');
SELECT try_cron_unschedule('cleanup_consumed_tokens');

DROP TABLE IF EXISTS consumed_tokens;
DROP TABLE IF EXISTS request_ids;
DROP TABLE IF EXISTS approvals;
