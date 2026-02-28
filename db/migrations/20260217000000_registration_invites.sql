-- +goose Up

CREATE TABLE registration_invites (
    id                    text        PRIMARY KEY CHECK (char_length(id) <= 255),
    user_id               uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    invite_code_hash      text        NOT NULL UNIQUE CHECK (char_length(invite_code_hash) <= 128),
    status                text        NOT NULL CHECK (status IN ('active', 'consumed', 'expired')),
    verification_attempts int         NOT NULL DEFAULT 0 CHECK (verification_attempts >= 0),
    expires_at            timestamptz NOT NULL,
    consumed_at           timestamptz,
    created_at            timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_registration_invites_user_status ON registration_invites (user_id, status);

-- pg_cron job to clean up expired invites
SELECT try_cron_schedule(
    'cleanup_expired_invites',
    '*/5 * * * *',
    $$UPDATE registration_invites SET status = 'expired' WHERE status = 'active' AND expires_at < now()$$
);

-- +goose Down
SELECT try_cron_unschedule('cleanup_expired_invites');
DROP TABLE IF EXISTS registration_invites;
