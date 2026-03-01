-- +goose Up
-- Tracks processed Stripe webhook event IDs for idempotency.
-- When a webhook handler fails, the event is NOT recorded, so Stripe's
-- retry mechanism can reprocess it. Events are recorded only after
-- successful processing.
CREATE TABLE IF NOT EXISTS stripe_webhook_events (
    event_id text PRIMARY KEY,      -- Stripe event ID (e.g. "evt_1234...")
    event_type text NOT NULL,       -- Stripe event type (e.g. "checkout.session.completed")
    processed_at timestamptz NOT NULL DEFAULT now()
);

-- Purge old events after 72 hours (Stripe retries for up to 72 hours).
-- A background job or periodic cleanup can use this index.
CREATE INDEX IF NOT EXISTS idx_stripe_webhook_events_processed_at
    ON stripe_webhook_events (processed_at);

-- +goose Down
DROP TABLE IF EXISTS stripe_webhook_events;
