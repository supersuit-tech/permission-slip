-- +goose Up

-- Prevent a connector from having both oauth2 and non-oauth2 required
-- credentials. The agent_connector_credentials table enforces one binding per
-- agent+connector (via unique index), so mixed auth types can never be
-- satisfied simultaneously — one side is always missing.
--
-- We enforce this at the schema level with a trigger that rejects inserts or
-- updates that would create a mixed-auth situation for a connector.

-- +goose StatementBegin
CREATE OR REPLACE FUNCTION check_no_mixed_auth_types()
RETURNS trigger AS $$
DECLARE
    new_is_oauth boolean;
    has_conflict boolean;
BEGIN
    new_is_oauth := (NEW.auth_type = 'oauth2');

    -- Check if any existing row for this connector has the opposite auth category.
    SELECT EXISTS (
        SELECT 1 FROM connector_required_credentials
        WHERE connector_id = NEW.connector_id
          AND (auth_type = 'oauth2') <> new_is_oauth
          -- Exclude the row being updated (if this is an UPDATE).
          -- Use OLD.auth_type so the exclusion matches the pre-update row.
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

CREATE TRIGGER trg_no_mixed_auth_types
    BEFORE INSERT OR UPDATE ON connector_required_credentials
    FOR EACH ROW
    EXECUTE FUNCTION check_no_mixed_auth_types();

-- Clean up any existing mixed-auth connectors by keeping only the majority
-- auth category. In practice this is a no-op since no mixed-auth connectors
-- have been created, but guards against edge cases.
-- +goose StatementBegin
DO $$
DECLARE
    r RECORD;
BEGIN
    -- Find connectors that have both oauth2 and non-oauth2 rows.
    FOR r IN
        SELECT connector_id,
               COUNT(*) FILTER (WHERE auth_type = 'oauth2') AS oauth_count,
               COUNT(*) FILTER (WHERE auth_type <> 'oauth2') AS non_oauth_count
        FROM connector_required_credentials
        GROUP BY connector_id
        HAVING COUNT(*) FILTER (WHERE auth_type = 'oauth2') > 0
           AND COUNT(*) FILTER (WHERE auth_type <> 'oauth2') > 0
    LOOP
        -- Keep the majority; on tie, keep non-oauth2 (static creds are simpler).
        IF r.oauth_count > r.non_oauth_count THEN
            DELETE FROM connector_required_credentials
            WHERE connector_id = r.connector_id AND auth_type <> 'oauth2';
        ELSE
            DELETE FROM connector_required_credentials
            WHERE connector_id = r.connector_id AND auth_type = 'oauth2';
        END IF;
    END LOOP;
END $$;
-- +goose StatementEnd

-- +goose Down
DROP TRIGGER IF EXISTS trg_no_mixed_auth_types ON connector_required_credentials;
DROP FUNCTION IF EXISTS check_no_mixed_auth_types();
