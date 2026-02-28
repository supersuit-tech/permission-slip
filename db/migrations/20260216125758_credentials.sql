-- +goose Up

CREATE TABLE credentials (
    id               text        PRIMARY KEY CHECK (char_length(id) <= 255),
    user_id          uuid        NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    service          text        NOT NULL CHECK (char_length(service) <= 255),
    label            text        CHECK (char_length(label) <= 255),
    vault_secret_id  uuid        NOT NULL,
    created_at       timestamptz NOT NULL DEFAULT now(),
    UNIQUE NULLS NOT DISTINCT (user_id, service, label)
);

CREATE INDEX idx_credentials_user_service ON credentials (user_id, service);

-- +goose Down
DROP TABLE IF EXISTS credentials;
