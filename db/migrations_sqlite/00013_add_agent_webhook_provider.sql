-- +goose Up

ALTER TABLE agents ADD COLUMN webhook_provider TEXT NOT NULL DEFAULT 'openclaw'
  CHECK (webhook_provider IN ('openclaw', 'grokbot'));

-- +goose Down

ALTER TABLE agents DROP COLUMN webhook_provider;
