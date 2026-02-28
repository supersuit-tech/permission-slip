-- +goose Up

CREATE TABLE agents (
    agent_id               text        PRIMARY KEY CHECK (char_length(agent_id) <= 255),
    public_key             text        NOT NULL CHECK (char_length(public_key) <= 1024),
    approver_id            uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    status                 text        NOT NULL CHECK (status IN ('pending', 'registered')),
    metadata               jsonb       CHECK (pg_column_size(metadata) <= 65536),
    confirmation_code_hash text        CHECK (char_length(confirmation_code_hash) <= 128),
    verification_attempts  int         NOT NULL DEFAULT 0 CHECK (verification_attempts >= 0),
    registration_ttl       int         CHECK (registration_ttl BETWEEN 60 AND 86400),
    expires_at             timestamptz,
    registered_at          timestamptz,
    created_at             timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_agents_approver_status ON agents (approver_id, status);

-- +goose Down
DROP TABLE IF EXISTS agents;
