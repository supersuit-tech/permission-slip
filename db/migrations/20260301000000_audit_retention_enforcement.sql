-- +goose Up

-- Track when a user's subscription was downgraded so the purge logic can
-- apply a grace period before enforcing the shorter retention window.
ALTER TABLE subscriptions ADD COLUMN downgraded_at timestamptz;

-- SQL function for plan-aware audit event cleanup, mirroring the Go-level
-- PurgeExpiredAuditEvents logic. During the 7-day downgrade grace period
-- the previous paid plan's 90-day retention is used.
--
-- This is scheduled via pg_cron for environments that support it (e.g.
-- Supabase). The Go background goroutine (startAuditPurge) continues to
-- run as well; both are idempotent and safe to run concurrently.

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION purge_expired_audit_events() RETURNS void LANGUAGE plpgsql AS $$
DECLARE
    pass1_count bigint;
    pass2_count bigint;
BEGIN
    -- Pass 1: Users with subscriptions.
    -- Grace period: if downgraded within last 7 days, use 90-day retention.
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

-- Schedule daily cleanup at 3:00 AM UTC.
SELECT try_cron_schedule(
    'purge_expired_audit_events',
    '0 3 * * *',
    'SELECT purge_expired_audit_events()'
);

-- +goose Down

SELECT try_cron_unschedule('purge_expired_audit_events');
DROP FUNCTION IF EXISTS purge_expired_audit_events();
ALTER TABLE subscriptions DROP COLUMN IF EXISTS downgraded_at;
