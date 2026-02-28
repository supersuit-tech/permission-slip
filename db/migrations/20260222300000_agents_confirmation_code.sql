-- +goose Up
-- Add a plain-text confirmation_code column to agents for dashboard display.
-- This stores the human-readable code (e.g. "XK7-M9P") temporarily while the
-- agent is in pending status. It is cleared when the agent verifies.
ALTER TABLE agents ADD COLUMN confirmation_code text CHECK (char_length(confirmation_code) <= 10);

-- +goose Down
ALTER TABLE agents DROP COLUMN IF EXISTS confirmation_code;
