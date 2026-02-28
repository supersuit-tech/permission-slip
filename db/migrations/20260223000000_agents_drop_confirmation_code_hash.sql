-- +goose Up
-- Remove the confirmation_code_hash column from agents. Confirmation codes are
-- short-lived (5-minute TTL), low-entropy (6 chars), and must be displayed in
-- plaintext on the dashboard anyway — hashing adds complexity without meaningful
-- security benefit. Verification now compares the submitted code directly against
-- the stored plaintext confirmation_code column.
ALTER TABLE agents DROP COLUMN IF EXISTS confirmation_code_hash;

-- +goose Down
ALTER TABLE agents ADD COLUMN confirmation_code_hash text CHECK (char_length(confirmation_code_hash) <= 128);
