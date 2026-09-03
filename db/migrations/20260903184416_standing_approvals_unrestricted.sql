-- +goose Up
ALTER TABLE standing_approvals
    ADD COLUMN unrestricted boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE standing_approvals
    DROP COLUMN unrestricted;
