-- +goose Up
-- Allow multiple OAuth connections per provider per user (e.g. two Google
-- accounts for different agents). The old unique constraint enforced one
-- connection per provider. Replace it with a plain index for query perf.

ALTER TABLE oauth_connections DROP CONSTRAINT IF EXISTS oauth_connections_user_id_provider_key;
CREATE INDEX IF NOT EXISTS idx_oauth_connections_user_provider ON oauth_connections (user_id, provider);

-- +goose Down
DROP INDEX IF EXISTS idx_oauth_connections_user_provider;
ALTER TABLE oauth_connections ADD CONSTRAINT oauth_connections_user_id_provider_key UNIQUE (user_id, provider);
