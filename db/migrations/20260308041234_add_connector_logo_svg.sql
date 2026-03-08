-- +goose Up
ALTER TABLE connectors ADD COLUMN logo_svg TEXT;

-- +goose Down
ALTER TABLE connectors DROP COLUMN logo_svg;
