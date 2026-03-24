-- +goose Up

-- BYOA (Bring Your Own App) support has been removed. Users who want to use
-- their own OAuth app can spin up the web server with their own environment
-- variables. Drop the table that stored per-user BYOA credentials.
--
-- Vault secrets referenced by client_id_vault_id and client_secret_vault_id
-- are orphaned but harmless — Supabase Vault will retain them until manually
-- cleaned up or the vault extension is reconfigured.
DROP TABLE IF EXISTS oauth_provider_configs;

-- +goose Down

-- Recreate the table with the same schema as the original migrations.
CREATE TABLE oauth_provider_configs (
    id                     text        PRIMARY KEY CHECK (char_length(id) <= 255),
    user_id                uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    provider               text        NOT NULL CHECK (char_length(provider) <= 255),
    client_id_vault_id     uuid        NOT NULL,
    client_secret_vault_id uuid        NOT NULL,
    created_at             timestamptz NOT NULL DEFAULT now(),
    updated_at             timestamptz NOT NULL DEFAULT now(),
    UNIQUE (user_id, provider)
);

CREATE INDEX idx_oauth_provider_configs_user ON oauth_provider_configs (user_id);
