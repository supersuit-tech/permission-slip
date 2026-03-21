-- +goose Up
ALTER TABLE subscriptions ADD COLUMN is_free_pro boolean NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE subscriptions DROP COLUMN is_free_pro;
