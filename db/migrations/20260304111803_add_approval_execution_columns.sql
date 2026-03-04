-- +goose Up
ALTER TABLE approvals
    ADD COLUMN execution_status text CHECK (execution_status IN ('pending', 'success', 'error')),
    ADD COLUMN execution_result jsonb,
    ADD COLUMN executed_at     timestamptz;

-- Enforce consistency: execution columns must be set together.
-- Either all NULL (not yet executed) or status + timestamp both present.
ALTER TABLE approvals
    ADD CONSTRAINT chk_execution_columns_consistent
    CHECK (
        (execution_status IS NULL AND executed_at IS NULL)
        OR (execution_status IS NOT NULL AND executed_at IS NOT NULL)
    );

-- +goose Down
ALTER TABLE approvals DROP CONSTRAINT IF EXISTS chk_execution_columns_consistent;
ALTER TABLE approvals
    DROP COLUMN IF EXISTS executed_at,
    DROP COLUMN IF EXISTS execution_result,
    DROP COLUMN IF EXISTS execution_status;
