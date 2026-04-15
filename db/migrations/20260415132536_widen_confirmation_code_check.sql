-- +goose Up
-- Widen the confirmation_code length check from 10 to 11 characters.
-- The security audit (#946) bumped confirmation code entropy from ~30 to ~50
-- bits by going from 6 → 10 characters. The plaintext stored here includes
-- a hyphen separator (XXXXX-XXXXX format), so the total length is 11.
ALTER TABLE agents DROP CONSTRAINT IF EXISTS agents_confirmation_code_check;
ALTER TABLE agents ADD CONSTRAINT agents_confirmation_code_check
    CHECK (char_length(confirmation_code) <= 11);

-- +goose Down
ALTER TABLE agents DROP CONSTRAINT IF EXISTS agents_confirmation_code_check;
ALTER TABLE agents ADD CONSTRAINT agents_confirmation_code_check
    CHECK (char_length(confirmation_code) <= 10);
