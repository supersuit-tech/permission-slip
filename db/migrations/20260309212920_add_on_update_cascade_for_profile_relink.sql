-- +goose Up

-- Add ON UPDATE CASCADE to all foreign keys referencing profiles(id).
-- This allows transparent profile re-linking when a returning user's
-- Supabase auth identity gets a new UUID for the same email address.
-- We can then UPDATE profiles SET id = $new WHERE id = $old and all
-- child rows follow automatically.

ALTER TABLE action_configurations
    DROP CONSTRAINT action_configurations_user_id_fkey,
    ADD CONSTRAINT action_configurations_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE agents
    DROP CONSTRAINT agents_approver_id_fkey,
    ADD CONSTRAINT agents_approver_id_fkey
        FOREIGN KEY (approver_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE approvals
    DROP CONSTRAINT approvals_approver_id_fkey,
    ADD CONSTRAINT approvals_approver_id_fkey
        FOREIGN KEY (approver_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE audit_events
    DROP CONSTRAINT audit_events_user_id_fkey,
    ADD CONSTRAINT audit_events_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE credentials
    DROP CONSTRAINT credentials_user_id_fkey,
    ADD CONSTRAINT credentials_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE expo_push_tokens
    DROP CONSTRAINT expo_push_tokens_user_id_fkey,
    ADD CONSTRAINT expo_push_tokens_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_user_id_fkey,
    ADD CONSTRAINT notification_preferences_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE oauth_connections
    DROP CONSTRAINT oauth_connections_user_id_fkey,
    ADD CONSTRAINT oauth_connections_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE oauth_provider_configs
    DROP CONSTRAINT oauth_provider_configs_user_id_fkey,
    ADD CONSTRAINT oauth_provider_configs_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE payment_method_transactions
    DROP CONSTRAINT payment_method_transactions_user_id_fkey,
    ADD CONSTRAINT payment_method_transactions_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE payment_methods
    DROP CONSTRAINT payment_methods_user_id_fkey,
    ADD CONSTRAINT payment_methods_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE push_subscriptions
    DROP CONSTRAINT push_subscriptions_user_id_fkey,
    ADD CONSTRAINT push_subscriptions_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE registration_invites
    DROP CONSTRAINT registration_invites_user_id_fkey,
    ADD CONSTRAINT registration_invites_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE standing_approvals
    DROP CONSTRAINT standing_approvals_user_id_fkey,
    ADD CONSTRAINT standing_approvals_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE subscriptions
    DROP CONSTRAINT subscriptions_user_id_fkey,
    ADD CONSTRAINT subscriptions_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

ALTER TABLE usage_periods
    DROP CONSTRAINT usage_periods_user_id_fkey,
    ADD CONSTRAINT usage_periods_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE ON UPDATE CASCADE;

-- Ensure the local-dev auth.users stub has an email column so the
-- email-based profile recovery query works in test environments.
-- In production Supabase, auth.users already has this column.
-- +goose StatementBegin
DO $$ BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_schema = 'auth' AND table_name = 'users' AND column_name = 'email'
    ) THEN
        ALTER TABLE auth.users ADD COLUMN email text;
        CREATE UNIQUE INDEX IF NOT EXISTS users_email_key ON auth.users (email);
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down

-- Revert all constraints back to ON UPDATE NO ACTION (the default).

ALTER TABLE action_configurations
    DROP CONSTRAINT action_configurations_user_id_fkey,
    ADD CONSTRAINT action_configurations_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE agents
    DROP CONSTRAINT agents_approver_id_fkey,
    ADD CONSTRAINT agents_approver_id_fkey
        FOREIGN KEY (approver_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE approvals
    DROP CONSTRAINT approvals_approver_id_fkey,
    ADD CONSTRAINT approvals_approver_id_fkey
        FOREIGN KEY (approver_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE audit_events
    DROP CONSTRAINT audit_events_user_id_fkey,
    ADD CONSTRAINT audit_events_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE credentials
    DROP CONSTRAINT credentials_user_id_fkey,
    ADD CONSTRAINT credentials_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE expo_push_tokens
    DROP CONSTRAINT expo_push_tokens_user_id_fkey,
    ADD CONSTRAINT expo_push_tokens_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE notification_preferences
    DROP CONSTRAINT notification_preferences_user_id_fkey,
    ADD CONSTRAINT notification_preferences_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE oauth_connections
    DROP CONSTRAINT oauth_connections_user_id_fkey,
    ADD CONSTRAINT oauth_connections_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE oauth_provider_configs
    DROP CONSTRAINT oauth_provider_configs_user_id_fkey,
    ADD CONSTRAINT oauth_provider_configs_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE payment_method_transactions
    DROP CONSTRAINT payment_method_transactions_user_id_fkey,
    ADD CONSTRAINT payment_method_transactions_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE payment_methods
    DROP CONSTRAINT payment_methods_user_id_fkey,
    ADD CONSTRAINT payment_methods_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE push_subscriptions
    DROP CONSTRAINT push_subscriptions_user_id_fkey,
    ADD CONSTRAINT push_subscriptions_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE registration_invites
    DROP CONSTRAINT registration_invites_user_id_fkey,
    ADD CONSTRAINT registration_invites_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE standing_approvals
    DROP CONSTRAINT standing_approvals_user_id_fkey,
    ADD CONSTRAINT standing_approvals_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE subscriptions
    DROP CONSTRAINT subscriptions_user_id_fkey,
    ADD CONSTRAINT subscriptions_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

ALTER TABLE usage_periods
    DROP CONSTRAINT usage_periods_user_id_fkey,
    ADD CONSTRAINT usage_periods_user_id_fkey
        FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE;

-- NOTE: We intentionally do NOT drop auth.users.email here.
-- In production Supabase, auth.users.email pre-exists (the UP block's
-- IF NOT EXISTS guard was false and we never touched it). Dropping it
-- would destroy Supabase authentication. In local dev, the column is
-- harmless to leave in place. A no-op is safer than an irreversible
-- production column drop.
