-- +goose Up

-- Remove confirmation-code / token columns from approvals (no longer needed
-- for the direct-execution approval flow).
ALTER TABLE approvals
    DROP COLUMN IF EXISTS confirmation_code_hash,
    DROP COLUMN IF EXISTS verification_attempts,
    DROP COLUMN IF EXISTS token_jti;

-- Drop consumed_tokens table and its cleanup cron job.
SELECT try_cron_unschedule('cleanup_consumed_tokens');
DROP TABLE IF EXISTS consumed_tokens;

-- +goose Down

-- Restore columns on approvals.
ALTER TABLE approvals
    ADD COLUMN confirmation_code_hash text CHECK (char_length(confirmation_code_hash) <= 128),
    ADD COLUMN verification_attempts  int NOT NULL DEFAULT 0 CHECK (verification_attempts >= 0),
    ADD COLUMN token_jti              text UNIQUE CHECK (char_length(token_jti) <= 255);

-- Restore consumed_tokens table.
CREATE TABLE consumed_tokens (
    jti         text        PRIMARY KEY CHECK (char_length(jti) <= 255),
    consumed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_consumed_tokens_consumed_at ON consumed_tokens (consumed_at);

SELECT try_cron_schedule(
    'cleanup_consumed_tokens',
    '0 * * * *',
    $$DELETE FROM consumed_tokens WHERE consumed_at < now() - interval '24 hours'$$
);
