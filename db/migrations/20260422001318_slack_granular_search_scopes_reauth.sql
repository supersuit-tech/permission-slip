-- +goose Up

-- The Slack connector now requests the granular search:read.public,
-- search:read.private, search:read.im, search:read.mpim, search:read.files
-- scopes instead of the legacy monolithic search:read. Slack's search.messages
-- endpoint rejects the legacy scope at runtime with invalid_arguments.
--
-- Existing connections only have the legacy search:read scope in their stored
-- scopes array, so they fail both the credentials_ready subset check (the
-- startup-time manifest upsert updates connector_required_credentials to the
-- granular scopes) and the actual Slack API call. Force re-authorization so
-- users get a token with the new scopes.
UPDATE oauth_connections
SET status = 'needs_reauth', updated_at = now()
WHERE provider = 'slack'
  AND status = 'active'
  AND NOT (ARRAY[
    'search:read.public',
    'search:read.private',
    'search:read.im',
    'search:read.mpim',
    'search:read.files'
  ]::text[] <@ scopes);

-- +goose Down

-- Not reversible: we cannot distinguish rows that were flipped by this
-- migration from those already in needs_reauth for other reasons.
