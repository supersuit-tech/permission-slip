-- +goose Up

ALTER TABLE oauth_provider_configs
    ADD COLUMN updated_at timestamptz NOT NULL DEFAULT now();

-- +goose Down

ALTER TABLE oauth_provider_configs
    DROP COLUMN IF EXISTS updated_at;
