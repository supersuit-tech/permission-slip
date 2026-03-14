-- +goose Up
-- Plan definitions have moved to config/plans.json (single source of truth).
-- The plans table is no longer needed — subscriptions.plan_id is now validated
-- at the application layer against the config-defined plans.

-- First, replace the purge function so it no longer JOINs the plans table.
-- Retention days are inlined matching config/plans.json values.
-- IMPORTANT: If retention days in config/plans.json are ever changed, a new
-- migration must update this function to match. The Go PurgeExpiredAuditEvents
-- in data_retention.go derives values dynamically from config, but this stored
-- function (used by pg_cron) uses hardcoded values.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION purge_expired_audit_events() RETURNS void
    LANGUAGE plpgsql
    SECURITY INVOKER
AS $$
DECLARE
    pass1_count bigint;
    pass2_count bigint;
BEGIN
    -- Pass 1: Users with subscriptions.
    -- Retention days mirror config/plans.json:
    --   free = 7 days, pay_as_you_go = 90 days.
    -- Grace period: if downgraded within last 7 days, use 90-day retention.
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
                   ELSE 7
               END
          END);
    GET DIAGNOSTICS pass1_count = ROW_COUNT;

    -- Pass 2: Users without subscriptions (defensive fallback, 7-day retention).
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

-- Normalize any legacy plan_id values to 'free' before dropping the table.
-- This prevents 500s for users with plan IDs not in config/plans.json.
UPDATE subscriptions SET plan_id = 'free'
WHERE plan_id NOT IN ('free', 'pay_as_you_go');

-- Now safe to drop the FK constraint and table.
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_plan_id_fkey;

DROP TABLE IF EXISTS plans;

-- Add a CHECK constraint to replace the FK as a write-time guard.
-- Only config-defined plan IDs are allowed.
ALTER TABLE subscriptions ADD CONSTRAINT subscriptions_plan_id_check
    CHECK (plan_id IN ('free', 'pay_as_you_go'));

-- +goose Down
-- Drop the CHECK constraint and re-create the plans table and FK constraint.
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_plan_id_check;
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
    ('free',          'Free',          1000, 3,    5,    5,    7,  0),
    ('pay_as_you_go', 'Pay As You Go', NULL, NULL, NULL, NULL, 90, 500)
ON CONFLICT (id) DO NOTHING;

ALTER TABLE subscriptions
    ADD CONSTRAINT subscriptions_plan_id_fkey FOREIGN KEY (plan_id) REFERENCES plans(id);

-- Restore the purge function that JOINs the plans table.
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
    JOIN plans p ON p.id = s.plan_id
    WHERE ae.user_id = s.user_id
      AND ae.created_at < now() - make_interval(days =>
          CASE WHEN s.downgraded_at IS NOT NULL
                    AND s.downgraded_at > now() - interval '7 days'
               THEN 90
               ELSE p.audit_retention_days
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
