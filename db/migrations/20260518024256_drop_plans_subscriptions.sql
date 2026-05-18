-- +goose Up
-- Self-hosted: remove subscription billing tables; keep Stripe customer IDs on profiles
-- for SetupIntents / stored payment methods (issue #1219).

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION purge_expired_audit_events() RETURNS void
    LANGUAGE plpgsql
    SECURITY INVOKER
AS $$
DECLARE
    deleted_count bigint;
BEGIN
    -- Single retention window for all users (default 90 days, matches AUDIT_RETENTION_DAYS in Go).
    DELETE FROM audit_events
     WHERE created_at < now() - interval '90 days';
    GET DIAGNOSTICS deleted_count = ROW_COUNT;

    IF deleted_count > 0 THEN
        RAISE LOG 'purge_expired_audit_events: deleted % rows', deleted_count;
    END IF;
END;
$$;
-- +goose StatementEnd

ALTER TABLE profiles ADD COLUMN IF NOT EXISTS stripe_customer_id text;

UPDATE profiles p
SET stripe_customer_id = s.stripe_customer_id
FROM subscriptions s
WHERE p.id = s.user_id
  AND s.stripe_customer_id IS NOT NULL
  AND p.stripe_customer_id IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_profiles_stripe_customer_id
    ON profiles (stripe_customer_id)
    WHERE stripe_customer_id IS NOT NULL;

DROP TABLE IF EXISTS stripe_webhook_events;
DROP TABLE IF EXISTS subscriptions;

-- +goose Down
-- Recreate subscription-era tables and restore the prior purge function.
-- Stripe customer IDs are copied back from profiles where present.

CREATE TABLE IF NOT EXISTS stripe_webhook_events (
    event_id text PRIMARY KEY,
    event_type text NOT NULL,
    processed_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_stripe_webhook_events_processed_at
    ON stripe_webhook_events (processed_at);

CREATE TABLE IF NOT EXISTS subscriptions (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL UNIQUE REFERENCES profiles(id) ON DELETE CASCADE,
    plan_id text NOT NULL DEFAULT 'free'
        CHECK (plan_id IN ('free', 'pay_as_you_go', 'free_pro')),
    status text NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'past_due', 'cancelled')),
    stripe_customer_id text,
    stripe_subscription_id text,
    current_period_start timestamptz NOT NULL DEFAULT date_trunc('month', now() AT TIME ZONE 'utc') AT TIME ZONE 'utc',
    current_period_end timestamptz NOT NULL DEFAULT date_trunc('month', now() AT TIME ZONE 'utc') AT TIME ZONE 'utc' + interval '1 month',
    downgraded_at timestamptz,
    quota_plan_id text,
    quota_entitlements_until timestamptz,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT subscriptions_quota_grace_pair_chk CHECK (
        (quota_plan_id IS NULL AND quota_entitlements_until IS NULL)
        OR (quota_plan_id IS NOT NULL AND quota_entitlements_until IS NOT NULL)
    )
);

CREATE INDEX IF NOT EXISTS idx_subscriptions_plan_id ON subscriptions (plan_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_stripe_customer_id
    ON subscriptions (stripe_customer_id)
    WHERE stripe_customer_id IS NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscriptions_stripe_subscription_id
    ON subscriptions (stripe_subscription_id)
    WHERE stripe_subscription_id IS NOT NULL;

INSERT INTO subscriptions (user_id, plan_id, stripe_customer_id)
SELECT id, 'free', stripe_customer_id
  FROM profiles;

DROP INDEX IF EXISTS idx_profiles_stripe_customer_id;
ALTER TABLE profiles DROP COLUMN IF EXISTS stripe_customer_id;

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION purge_expired_audit_events() RETURNS void
    LANGUAGE plpgsql
    SECURITY INVOKER
AS $$
DECLARE
    pass1_count bigint;
    pass2_count bigint;
BEGIN
    DELETE FROM audit_events ae
    USING subscriptions s
    WHERE ae.user_id = s.user_id
      AND ae.created_at < now() - make_interval(days =>
          CASE WHEN s.downgraded_at IS NOT NULL
                    AND s.downgraded_at > now() - interval '7 days'
               THEN 90
               ELSE CASE s.plan_id
                   WHEN 'free' THEN 7
                   WHEN 'pay_as_you_go' THEN 90
                   WHEN 'free_pro' THEN 90
                   ELSE 7
               END
          END);
    GET DIAGNOSTICS pass1_count = ROW_COUNT;

    DELETE FROM audit_events ae
    WHERE NOT EXISTS (SELECT 1 FROM subscriptions s WHERE s.user_id = ae.user_id)
      AND ae.created_at < now() - interval '7 days';
    GET DIAGNOSTICS pass2_count = ROW_COUNT;

    IF pass1_count + pass2_count > 0 THEN
        RAISE LOG 'purge_expired_audit_events: deleted % rows (pass1=%, pass2=%)',
            pass1_count + pass2_count, pass1_count, pass2_count;
    END IF;
END;
$$;
-- +goose StatementEnd
