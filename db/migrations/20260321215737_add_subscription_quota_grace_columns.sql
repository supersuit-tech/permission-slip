-- +goose Up

-- Quota grace after downgrade to free: paid plan limits apply until quota_entitlements_until.
ALTER TABLE subscriptions
    ADD COLUMN quota_plan_id text,
    ADD COLUMN quota_entitlements_until timestamptz;

ALTER TABLE subscriptions
    ADD CONSTRAINT subscriptions_quota_grace_pair_chk CHECK (
        (quota_plan_id IS NULL AND quota_entitlements_until IS NULL)
        OR (quota_plan_id IS NOT NULL AND quota_entitlements_until IS NOT NULL)
    );

-- +goose Down

ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_quota_grace_pair_chk;

ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS quota_plan_id,
    DROP COLUMN IF EXISTS quota_entitlements_until;
