-- +goose Up
ALTER TABLE standing_approvals ADD COLUMN resource_details TEXT;
ALTER TABLE standing_approval_requests ADD COLUMN resource_details TEXT;

-- +goose Down
ALTER TABLE standing_approvals DROP COLUMN resource_details;
ALTER TABLE standing_approval_requests DROP COLUMN resource_details;
