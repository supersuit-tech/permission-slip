-- +goose Up
ALTER TABLE standing_approval_requests DROP COLUMN IF EXISTS max_executions;
ALTER TABLE standing_approval_requests DROP COLUMN IF EXISTS expires_in_seconds;

-- +goose Down
ALTER TABLE standing_approval_requests
    ADD COLUMN max_executions int CHECK (max_executions IS NULL OR max_executions > 0),
    ADD COLUMN expires_in_seconds int CHECK (expires_in_seconds IS NULL OR expires_in_seconds > 0);
