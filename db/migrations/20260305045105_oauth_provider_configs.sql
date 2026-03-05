-- +goose Up

CREATE TABLE oauth_provider_configs (
    id                     text        PRIMARY KEY CHECK (char_length(id) <= 255),
    user_id                uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    provider               text        NOT NULL CHECK (char_length(provider) <= 255),
    client_id_vault_id     uuid        NOT NULL,
    client_secret_vault_id uuid        NOT NULL,
    created_at             timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, provider)
);

CREATE INDEX idx_oauth_provider_configs_user ON oauth_provider_configs (user_id);

-- +goose Down
DROP TABLE IF EXISTS oauth_provider_configs;
