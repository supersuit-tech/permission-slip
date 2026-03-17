-- +goose Up

-- Make expires_at nullable (NULL = no expiry, lasts until revoked).
ALTER TABLE standing_approvals ALTER COLUMN expires_at DROP NOT NULL;

-- Drop the 90-day maximum duration constraint.
-- The constraint name is auto-generated; find it by definition.
-- +goose StatementBegin
DO $$
DECLARE
    constraint_name text;
BEGIN
    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'standing_approvals'::regclass
      AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%expires_at - starts_at%90 days%';
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE standing_approvals DROP CONSTRAINT %I', constraint_name);
    ELSE
        RAISE EXCEPTION 'Could not find 90-day duration CHECK constraint on standing_approvals – migration cannot proceed safely';
    END IF;
END $$;
-- +goose StatementEnd

-- Drop the expires_at >= starts_at constraint since expires_at can now be NULL.
-- Re-add it as a conditional: only enforce when expires_at IS NOT NULL.
-- +goose StatementBegin
DO $$
DECLARE
    constraint_name text;
BEGIN
    SELECT conname INTO constraint_name
    FROM pg_constraint
    WHERE conrelid = 'standing_approvals'::regclass
      AND contype = 'c'
      AND pg_get_constraintdef(oid) LIKE '%expires_at >= starts_at%';
    IF constraint_name IS NOT NULL THEN
        EXECUTE format('ALTER TABLE standing_approvals DROP CONSTRAINT %I', constraint_name);
    ELSE
        RAISE EXCEPTION 'Could not find expires_at >= starts_at CHECK constraint on standing_approvals – migration cannot proceed safely';
    END IF;
END $$;
-- +goose StatementEnd

ALTER TABLE standing_approvals ADD CONSTRAINT standing_approvals_expires_at_after_starts_at
    CHECK (expires_at IS NULL OR expires_at >= starts_at);

-- +goose Down

-- Restore NOT NULL.
-- Refuse to roll back if permanent approvals exist — fabricating expiry dates would silently corrupt data.
-- +goose StatementBegin
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM standing_approvals WHERE expires_at IS NULL) THEN
        RAISE EXCEPTION 'Cannot roll back: standing approvals with no expiry (expires_at IS NULL) exist. Revoke or delete them first to avoid data corruption.';
    END IF;
END $$;
-- +goose StatementEnd
ALTER TABLE standing_approvals ALTER COLUMN expires_at SET NOT NULL;

-- Drop the conditional constraint and restore the original pair.
ALTER TABLE standing_approvals DROP CONSTRAINT IF EXISTS standing_approvals_expires_at_after_starts_at;
ALTER TABLE standing_approvals ADD CHECK (expires_at >= starts_at);
ALTER TABLE standing_approvals ADD CHECK (expires_at - starts_at <= interval '90 days');
