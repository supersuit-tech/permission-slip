-- +goose Up

-- plans: defines available pricing tiers and their resource limits.
CREATE TABLE plans (
    id                          text        PRIMARY KEY,
    name                        text        NOT NULL,
    max_requests_per_month      int,            -- NULL = unlimited
    max_agents                  int,            -- NULL = unlimited
    max_standing_approvals      int,            -- NULL = unlimited (active)
    max_credentials             int,            -- NULL = unlimited
    audit_retention_days        int         NOT NULL,
    price_per_request_millicents int        NOT NULL DEFAULT 0,  -- 1 millicent = $0.00001; $0.005 = 500 millicents
    created_at                  timestamptz NOT NULL DEFAULT now()
);

-- Seed the two plan tiers.
-- NOTE: The free plan limit below (1000) is stale — the authoritative value
-- is now in config/plans.json (currently 250). This migration is historical;
-- the plans table was dropped in 20260314131844_drop_plans_table.sql.
INSERT INTO plans (id, name, max_requests_per_month, max_agents, max_standing_approvals, max_credentials, audit_retention_days, price_per_request_millicents)
VALUES
    ('free',          'Free',          1000, 3,    5,    5,    7,  0),
    ('pay_as_you_go', 'Pay As You Go', NULL, NULL, NULL, NULL, 90, 500);

-- subscriptions: links a user to their current plan.
CREATE TABLE subscriptions (
    id                      uuid        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                 uuid        NOT NULL UNIQUE REFERENCES profiles(id) ON DELETE CASCADE,
    plan_id                 text        NOT NULL REFERENCES plans(id),
    status                  text        NOT NULL DEFAULT 'active'
                                        CHECK (status IN ('active', 'past_due', 'cancelled')),
    stripe_customer_id      text,           -- NULL for free-tier users
    stripe_subscription_id  text,           -- NULL for free-tier users
    current_period_start    timestamptz NOT NULL DEFAULT date_trunc('month', now()),
    current_period_end      timestamptz NOT NULL DEFAULT (date_trunc('month', now()) + interval '1 month'),
    created_at              timestamptz NOT NULL DEFAULT now(),
    updated_at              timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_subscriptions_plan_id ON subscriptions (plan_id);

-- Auto-create free subscriptions for all existing users.
INSERT INTO subscriptions (user_id, plan_id)
SELECT id, 'free' FROM profiles;

-- +goose Down

DROP TABLE IF EXISTS subscriptions;
DROP TABLE IF EXISTS plans;
