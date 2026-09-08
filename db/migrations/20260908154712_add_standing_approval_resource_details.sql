-- +goose Up
ALTER TABLE standing_approvals
    ADD COLUMN resource_details JSONB;

ALTER TABLE standing_approval_requests
    ADD COLUMN resource_details JSONB;

-- +goose Down
ALTER TABLE standing_approvals
    DROP COLUMN IF EXISTS resource_details;

ALTER TABLE standing_approval_requests
    DROP COLUMN IF EXISTS resource_details;
