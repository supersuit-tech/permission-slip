-- +goose Up

ALTER TABLE agents ADD COLUMN webhook_url text CHECK (webhook_url IS NULL OR char_length(webhook_url) <= 2048);
ALTER TABLE agents ADD COLUMN webhook_token_vault_id text;

-- +goose Down

ALTER TABLE agents DROP COLUMN IF EXISTS webhook_token_vault_id;
ALTER TABLE agents DROP COLUMN IF EXISTS webhook_url;
