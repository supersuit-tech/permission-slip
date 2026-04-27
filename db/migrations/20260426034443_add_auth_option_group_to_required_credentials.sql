-- +goose Up

-- Add auth_option_group to connector_required_credentials.
-- Credentials sharing the same non-null group are mutually exclusive
-- alternatives (e.g. "connect via OAuth OR PAT"). The user picks one;
-- the agent_connector_credentials binding satisfies whichever they chose.
ALTER TABLE connector_required_credentials
    ADD COLUMN auth_option_group text CHECK (char_length(auth_option_group) <= 255);

-- Update the mixed-auth trigger to allow mixing oauth2 and non-oauth2
-- within the same auth_option_group. Rows in the same group are
-- alternatives, so the "one binding per agent+connector" invariant holds.
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_no_mixed_auth_types()
RETURNS trigger AS $$
DECLARE
    new_is_oauth boolean;
    has_conflict boolean;
BEGIN
    new_is_oauth := (NEW.auth_type = 'oauth2');

    SELECT EXISTS (
        SELECT 1 FROM connector_required_credentials
        WHERE connector_id = NEW.connector_id
          AND (auth_type = 'oauth2') <> new_is_oauth
          -- Rows in the same non-null auth_option_group are alternatives, not a conflict.
          AND NOT (
              NEW.auth_option_group IS NOT NULL
              AND auth_option_group IS NOT DISTINCT FROM NEW.auth_option_group
          )
          -- Exclude the row being updated (UPDATE path).
          AND NOT (
              connector_id = NEW.connector_id
              AND service = NEW.service
              AND auth_type = CASE WHEN TG_OP = 'UPDATE' THEN OLD.auth_type
                                   ELSE NEW.auth_type END
          )
    ) INTO has_conflict;

    IF has_conflict THEN
        RAISE EXCEPTION
            'connector % cannot mix oauth2 and non-oauth2 auth types outside of an auth_option_group',
            NEW.connector_id
        USING ERRCODE = 'check_violation';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd

-- +goose Down
ALTER TABLE connector_required_credentials
    DROP COLUMN IF EXISTS auth_option_group;

-- Restore the original trigger (no auth_option_group awareness).
-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_no_mixed_auth_types()
RETURNS trigger AS $$
DECLARE
    new_is_oauth boolean;
    has_conflict boolean;
BEGIN
    new_is_oauth := (NEW.auth_type = 'oauth2');

    SELECT EXISTS (
        SELECT 1 FROM connector_required_credentials
        WHERE connector_id = NEW.connector_id
          AND (auth_type = 'oauth2') <> new_is_oauth
          AND NOT (
              connector_id = NEW.connector_id
              AND service = NEW.service
              AND auth_type = CASE WHEN TG_OP = 'UPDATE' THEN OLD.auth_type
                                   ELSE NEW.auth_type END
          )
    ) INTO has_conflict;

    IF has_conflict THEN
        RAISE EXCEPTION
            'connector % cannot mix oauth2 and non-oauth2 auth types',
            NEW.connector_id
        USING ERRCODE = 'check_violation';
    END IF;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
-- +goose StatementEnd
