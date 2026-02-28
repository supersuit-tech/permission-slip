-- +goose Up

-- server_config stores key-value pairs for server-wide configuration that must
-- persist across restarts (e.g. auto-generated VAPID keys). Using a simple
-- key-value table avoids creating a new table for each config need.
CREATE TABLE server_config (
    key   text PRIMARY KEY,
    value text NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- +goose Down
DROP TABLE IF EXISTS server_config;
