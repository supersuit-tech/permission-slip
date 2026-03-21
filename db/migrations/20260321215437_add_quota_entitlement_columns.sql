-- +goose Up
-- Add columns to track paid quota entitlements after downgrade.
-- quota_plan_id: the plan whose quotas still apply (e.g. 'pay_as_you_go')
-- quota_entitlements_until: when the quota grace period ends (snapshot of current_period_end at downgrade time)
ALTER TABLE subscriptions
    ADD COLUMN quota_plan_id TEXT,
    ADD COLUMN quota_entitlements_until TIMESTAMPTZ;

-- +goose Down
ALTER TABLE subscriptions
    DROP COLUMN IF EXISTS quota_entitlements_until,
    DROP COLUMN IF EXISTS quota_plan_id;
