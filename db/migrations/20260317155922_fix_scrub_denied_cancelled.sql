-- +goose Up

-- Fix: scrub_sensitive_execution_data() previously required executed_at IS NOT NULL,
-- which meant denied and cancelled approvals (whose executed_at is always NULL) were
-- never scrubbed. Use per-status explicit conditions to pick the correct timestamp.

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
    -- Use per-status conditions so each status checks its own resolution timestamp.
    -- For approved: require executed_at IS NOT NULL to avoid scrubbing in-flight executions.
    -- For denied/cancelled: use denied_at/cancelled_at respectively.
    UPDATE approvals
    SET execution_result = NULL,
        action = action - 'parameters'
    WHERE (execution_result IS NOT NULL OR action ? 'parameters')
      AND (
        (status = 'approved'   AND executed_at  IS NOT NULL AND executed_at  < now() - interval '30 minutes')
        OR (status = 'denied'    AND denied_at    IS NOT NULL AND denied_at    < now() - interval '30 minutes')
        OR (status = 'cancelled' AND cancelled_at IS NOT NULL AND cancelled_at < now() - interval '30 minutes')
      );
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

-- +goose Down

-- Restore original function that requires executed_at IS NOT NULL.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION scrub_sensitive_execution_data() RETURNS void
    LANGUAGE plpgsql
    SECURITY INVOKER
AS $$
DECLARE
    approvals_count bigint;
    executions_count bigint;
BEGIN
    UPDATE approvals
    SET execution_result = NULL,
        action = action - 'parameters'
    WHERE executed_at IS NOT NULL
      AND executed_at < now() - interval '30 minutes'
      AND status IN ('approved', 'denied', 'cancelled')
      AND (execution_result IS NOT NULL
           OR action ? 'parameters');
    GET DIAGNOSTICS approvals_count = ROW_COUNT;

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
