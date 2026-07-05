-- +goose Up
ALTER TABLE standing_approval_requests DROP COLUMN max_executions;
ALTER TABLE standing_approval_requests DROP COLUMN expires_in_seconds;

-- +goose Down
ALTER TABLE standing_approval_requests ADD COLUMN max_executions INTEGER;
ALTER TABLE standing_approval_requests ADD COLUMN expires_in_seconds INTEGER;
