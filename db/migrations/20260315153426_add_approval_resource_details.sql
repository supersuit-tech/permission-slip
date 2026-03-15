-- +goose Up
ALTER TABLE approvals ADD COLUMN resource_details JSONB;

-- +goose Down
ALTER TABLE approvals DROP COLUMN IF EXISTS resource_details;
