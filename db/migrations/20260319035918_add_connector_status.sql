-- +goose Up
ALTER TABLE connectors ADD COLUMN status text NOT NULL DEFAULT 'untested'
    CHECK (status IN ('tested', 'early_preview', 'untested'));

-- +goose Down
ALTER TABLE connectors DROP COLUMN status;
