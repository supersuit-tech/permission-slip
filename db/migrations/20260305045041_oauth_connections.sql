-- +goose Up

CREATE TABLE oauth_connections (
    id                     text        PRIMARY KEY CHECK (char_length(id) <= 255),
    user_id                uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    provider               text        NOT NULL CHECK (char_length(provider) <= 255),
    access_token_vault_id  uuid        NOT NULL,
    refresh_token_vault_id uuid,
    scopes                 text[]      NOT NULL DEFAULT '{}',
    token_expiry           timestamptz,
    status                 text        NOT NULL DEFAULT 'active'
                           CHECK (status IN ('active', 'needs_reauth', 'revoked')),
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, provider)
);

CREATE INDEX idx_oauth_connections_user ON oauth_connections (user_id);
CREATE INDEX idx_oauth_connections_status ON oauth_connections (user_id, status);

-- +goose Down
DROP TABLE IF EXISTS oauth_connections;
