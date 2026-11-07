-- +goose Up
ALTER TABLE approvals ADD COLUMN wake_callback_url TEXT CHECK (wake_callback_url IS NULL OR length(wake_callback_url) <= 2048);

-- +goose Down
ALTER TABLE approvals DROP COLUMN wake_callback_url;
