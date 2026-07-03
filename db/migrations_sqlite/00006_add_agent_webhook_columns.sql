-- +goose Up

ALTER TABLE agents ADD COLUMN webhook_url TEXT CHECK (webhook_url IS NULL OR length(webhook_url) <= 2048);
ALTER TABLE agents ADD COLUMN webhook_token_vault_id TEXT;

-- +goose Down

-- SQLite cannot DROP COLUMN in all versions; tests recreate from migrations.
ALTER TABLE agents DROP COLUMN webhook_token_vault_id;
ALTER TABLE agents DROP COLUMN webhook_url;
