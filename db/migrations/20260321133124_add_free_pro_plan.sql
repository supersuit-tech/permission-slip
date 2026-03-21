-- +goose Up
-- Comped "free pro" tier: unlimited usage, no per-request billing (plan enforced in app + Stripe reporting).

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

ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_plan_id_check;
ALTER TABLE subscriptions ADD CONSTRAINT subscriptions_plan_id_check
    CHECK (plan_id IN ('free', 'pay_as_you_go', 'free_pro'));

-- +goose Down
ALTER TABLE subscriptions DROP CONSTRAINT IF EXISTS subscriptions_plan_id_check;
ALTER TABLE subscriptions ADD CONSTRAINT subscriptions_plan_id_check
    CHECK (plan_id IN ('free', 'pay_as_you_go'));

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
