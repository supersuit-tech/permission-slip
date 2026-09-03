-- +goose Up
ALTER TABLE standing_approvals ADD COLUMN unrestricted INTEGER NOT NULL DEFAULT 0;

-- +goose Down
ALTER TABLE standing_approvals DROP COLUMN unrestricted;
