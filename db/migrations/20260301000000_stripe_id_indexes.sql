-- +goose Up
-- Add indexes for looking up subscriptions by Stripe IDs (used by webhook handlers).
-- These columns are nullable (free-tier users have no Stripe data), so we use
-- partial indexes to keep the index small and efficient.
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_stripe_customer_id
    ON subscriptions (stripe_customer_id)
    WHERE stripe_customer_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_stripe_subscription_id
    ON subscriptions (stripe_subscription_id)
    WHERE stripe_subscription_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_subscriptions_stripe_subscription_id;
DROP INDEX IF EXISTS idx_subscriptions_stripe_customer_id;
