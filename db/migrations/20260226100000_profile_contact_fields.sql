-- +goose Up
-- Add optional contact fields to profiles so users can opt in to
-- notification channels (email, SMS). Both are nullable — users who
-- haven't configured a channel simply have NULL.

ALTER TABLE profiles
    ADD COLUMN email text,
    ADD COLUMN phone text;

-- Email format: basic RFC-5321 local@domain check.
-- Intentionally permissive — full RFC-5322 compliance is impractical in a
-- CHECK constraint. Downstream senders validate more strictly.
ALTER TABLE profiles
    ADD CONSTRAINT profiles_email_format
    CHECK (email ~* '^[^@\s]+@[^@\s]+\.[^@\s]+$');

-- Phone format: E.164 (+ followed by 1–15 digits).
ALTER TABLE profiles
    ADD CONSTRAINT profiles_phone_e164
    CHECK (phone ~ '^\+[1-9][0-9]{0,14}$');

-- +goose Down
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS profiles_phone_e164;
ALTER TABLE profiles DROP CONSTRAINT IF EXISTS profiles_email_format;
ALTER TABLE profiles DROP COLUMN IF EXISTS phone;
ALTER TABLE profiles DROP COLUMN IF EXISTS email;
