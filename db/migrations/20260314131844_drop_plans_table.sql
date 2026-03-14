-- +goose Up
-- Plan definitions have moved to config/plans.json (single source of truth).
-- The plans table is no longer needed — subscriptions.plan_id is now validated
-- at the application layer against the config-defined plans.

-- Drop the FK constraint on subscriptions.plan_id first.
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_plan_id_fkey;

-- Drop the plans table.
DROP TABLE IF EXISTS plans;

-- +goose Down
-- Re-create the plans table and FK constraint.
CREATE TABLE IF NOT EXISTS plans (
    id                          text PRIMARY KEY,
    name                        text NOT NULL,
    max_requests_per_month      int,
    max_agents                  int,
    max_standing_approvals      int,
    max_credentials             int,
    audit_retention_days        int NOT NULL,
    price_per_request_millicents int NOT NULL DEFAULT 0,
    created_at                  timestamptz NOT NULL DEFAULT now()
);

INSERT INTO plans (id, name, max_requests_per_month, max_agents, max_standing_approvals, max_credentials, audit_retention_days, price_per_request_millicents)
VALUES
    ('free',          'Free',          250, 3,    5,    5,    7,  0),
    ('pay_as_you_go', 'Pay As You Go', NULL, NULL, NULL, NULL, 90, 500)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE subscriptions
    ADD CONSTRAINT subscriptions_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES plans(id);
