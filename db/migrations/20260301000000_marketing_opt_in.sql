-- +goose Up
-- Add marketing opt-in preference to profiles.
-- Defaults to false (opt-in, not opt-out).

ALTER TABLE profiles
  ADD COLUMN marketing_opt_in BOOLEAN NOT NULL DEFAULT false;

-- +goose Down
ALTER TABLE profiles DROP COLUMN IF EXISTS marketing_opt_in;
