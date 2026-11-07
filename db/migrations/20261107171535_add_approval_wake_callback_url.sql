-- +goose Up
ALTER TABLE approvals ADD COLUMN wake_callback_url text CHECK (wake_callback_url IS NULL OR char_length(wake_callback_url) <= 2048);

-- +goose Down
ALTER TABLE approvals DROP COLUMN IF EXISTS wake_callback_url;
