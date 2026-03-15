-- +goose Up

-- SQL function to scrub sensitive execution data older than 30 minutes.
-- Targets three columns across two tables:
--   1. approvals.execution_result → NULL
--   2. approvals.action → {"type":"<original_type>"} (parameters stripped)
--   3. standing_approval_executions.parameters → NULL
--
-- Only scrubs resolved approvals (approved/denied/cancelled) or executed ones.
-- Pending approvals keep their action parameters so approvers can review them.
-- Idempotent: WHERE clauses skip already-scrubbed rows.

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION scrub_sensitive_execution_data() RETURNS void
    LANGUAGE plpgsql
    SECURITY INVOKER
AS $$
DECLARE
    approvals_count bigint;
    executions_count bigint;
BEGIN
    -- Scrub approvals: NULL out execution_result, strip action to type-only.
    -- Only target resolved approvals with executed_at older than 30 minutes.
    UPDATE approvals
    SET execution_result = NULL,
        action = jsonb_build_object('type', action->>'type')
    WHERE executed_at IS NOT NULL
      AND executed_at < now() - interval '30 minutes'
      AND status IN ('approved', 'denied', 'cancelled')
      AND (execution_result IS NOT NULL
           OR action != jsonb_build_object('type', action->>'type'));
    GET DIAGNOSTICS approvals_count = ROW_COUNT;

    -- Scrub standing_approval_executions: NULL out parameters.
    UPDATE standing_approval_executions
    SET parameters = NULL
    WHERE executed_at < now() - interval '30 minutes'
      AND parameters IS NOT NULL;
    GET DIAGNOSTICS executions_count = ROW_COUNT;

    IF approvals_count + executions_count > 0 THEN
        RAISE LOG 'scrub_sensitive_execution_data: scrubbed % approvals, % standing_approval_executions',
            approvals_count, executions_count;
    END IF;
END;
$$;
-- +goose StatementEnd

-- Schedule every 5 minutes for redundancy alongside the Go background ticker.
SELECT try_cron_schedule(
    'scrub_sensitive_execution_data',
    '*/5 * * * *',
    'SELECT scrub_sensitive_execution_data()'
);

-- +goose Down

SELECT try_cron_unschedule('scrub_sensitive_execution_data');
DROP FUNCTION IF EXISTS scrub_sensitive_execution_data();
