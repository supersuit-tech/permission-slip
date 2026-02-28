-- +goose Up
-- Add request_id to standing_approval_executions for idempotency on the
-- agent-facing POST /v1/actions/execute endpoint. The unique index prevents
-- the same agent request from being executed twice against a given standing
-- approval.

ALTER TABLE standing_approval_executions
    ADD COLUMN request_id text;

CREATE UNIQUE INDEX idx_sa_executions_request_id
    ON standing_approval_executions (standing_approval_id, request_id)
    WHERE request_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_sa_executions_request_id;
ALTER TABLE standing_approval_executions DROP COLUMN IF EXISTS request_id;
