-- +goose Up

-- Revert of 20260422001318_slack_granular_search_scopes_reauth.sql.
--
-- That migration assumed the granular search:read.{public,private,im,mpim,files}
-- scopes satisfied Slack's search.messages endpoint. They do not: those scopes
-- only work with the newer Real-time Search API (assistant.search.context).
-- search.messages still requires the legacy monolithic search:read scope, and
-- tokens that only carry the granular scopes are rejected with
-- invalid_arguments — surfacing as 502 upstream_error for slack.search_messages
-- on every affected instance.
--
-- The connector now requests the legacy search:read user scope (and no longer
-- requests the granular search:read.* scopes). Force re-authorization for any
-- Slack connection whose stored scopes do not already include search:read so
-- users pick up a working token.
UPDATE oauth_connections
SET status = 'needs_reauth', updated_at = now()
WHERE provider = 'slack'
  AND status = 'active'
  AND NOT ('search:read' = ANY (scopes));

-- +goose Down

-- Not reversible: we cannot distinguish rows flipped by this migration from
-- those already in needs_reauth for other reasons.
