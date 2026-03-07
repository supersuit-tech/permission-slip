-- +goose Up
ALTER TABLE connector_actions ADD COLUMN requires_payment_method BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE connector_actions DROP COLUMN IF EXISTS requires_payment_method;
