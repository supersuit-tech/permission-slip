-- +goose Up

ALTER TABLE agents ADD COLUMN webhook_provider text NOT NULL DEFAULT 'openclaw';
ALTER TABLE agents ADD CONSTRAINT agents_webhook_provider_check
  CHECK (webhook_provider IN ('openclaw', 'grokbot'));

-- +goose Down

ALTER TABLE agents DROP CONSTRAINT IF EXISTS agents_webhook_provider_check;
ALTER TABLE agents DROP COLUMN IF EXISTS webhook_provider;
