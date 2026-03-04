-- +goose Up
ALTER TABLE approvals
    ADD COLUMN execution_status text CHECK (execution_status IN ('pending', 'success', 'error')),
    ADD COLUMN execution_result jsonb,
    ADD COLUMN executed_at     timestamptz;

-- +goose Down
ALTER TABLE approvals
    DROP COLUMN IF EXISTS executed_at,
    DROP COLUMN IF EXISTS execution_result,
    DROP COLUMN IF EXISTS execution_status;
