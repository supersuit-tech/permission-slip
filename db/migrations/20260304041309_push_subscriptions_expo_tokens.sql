-- +goose Up
-- Extend push_subscriptions to support Expo push tokens (mobile-push channel)
-- alongside the existing Web Push subscriptions.

-- Add channel column: 'web-push' (default) or 'mobile-push'.
ALTER TABLE push_subscriptions
    ADD COLUMN channel text NOT NULL DEFAULT 'web-push';

ALTER TABLE push_subscriptions
    ADD CONSTRAINT push_subscriptions_channel_check
    CHECK (channel IN ('web-push', 'mobile-push'));

-- Add expo_token column for storing Expo push tokens (mobile-push only).
ALTER TABLE push_subscriptions
    ADD COLUMN expo_token text;

-- Make p256dh and auth nullable — they are only required for web-push.
ALTER TABLE push_subscriptions ALTER COLUMN p256dh DROP NOT NULL;
ALTER TABLE push_subscriptions ALTER COLUMN auth DROP NOT NULL;

-- Make endpoint nullable — it is only required for web-push.
-- Drop the existing unique constraint first, then re-add as a partial index.
ALTER TABLE push_subscriptions DROP CONSTRAINT push_subscriptions_user_id_endpoint_key;
ALTER TABLE push_subscriptions ALTER COLUMN endpoint DROP NOT NULL;

-- Re-add the web-push deduplication index (only on rows with an endpoint).
CREATE UNIQUE INDEX idx_push_subscriptions_user_endpoint
    ON push_subscriptions (user_id, endpoint)
    WHERE endpoint IS NOT NULL;

-- Deduplication index for Expo tokens: one token per user.
CREATE UNIQUE INDEX idx_push_subscriptions_user_expo_token
    ON push_subscriptions (user_id, expo_token)
    WHERE expo_token IS NOT NULL;

-- Enforce field requirements per channel:
-- web-push rows must have endpoint, p256dh, and auth.
ALTER TABLE push_subscriptions
    ADD CONSTRAINT push_subscriptions_web_push_fields
    CHECK (channel != 'web-push' OR (endpoint IS NOT NULL AND p256dh IS NOT NULL AND auth IS NOT NULL));

-- mobile-push rows must have expo_token.
ALTER TABLE push_subscriptions
    ADD CONSTRAINT push_subscriptions_mobile_push_fields
    CHECK (channel != 'mobile-push' OR expo_token IS NOT NULL);

-- +goose Down
ALTER TABLE push_subscriptions DROP CONSTRAINT IF EXISTS push_subscriptions_mobile_push_fields;
ALTER TABLE push_subscriptions DROP CONSTRAINT IF EXISTS push_subscriptions_web_push_fields;
DROP INDEX IF EXISTS idx_push_subscriptions_user_expo_token;
DROP INDEX IF EXISTS idx_push_subscriptions_user_endpoint;
ALTER TABLE push_subscriptions ALTER COLUMN endpoint SET NOT NULL;
ALTER TABLE push_subscriptions ADD CONSTRAINT push_subscriptions_user_id_endpoint_key UNIQUE (user_id, endpoint);
ALTER TABLE push_subscriptions ALTER COLUMN auth SET NOT NULL;
ALTER TABLE push_subscriptions ALTER COLUMN p256dh SET NOT NULL;
ALTER TABLE push_subscriptions DROP COLUMN IF EXISTS expo_token;
ALTER TABLE push_subscriptions DROP CONSTRAINT IF EXISTS push_subscriptions_channel_check;
ALTER TABLE push_subscriptions DROP COLUMN IF EXISTS channel;
