-- +goose Up

-- Slack OAuth is now user-token-only (issue #772). Existing connections may
-- still store a bot token as the primary vault secret; force re-authorization.
UPDATE oauth_connections
SET status = 'needs_reauth', updated_at = now()
WHERE provider = 'slack' AND status = 'active';

-- +goose Down

-- Not reversible: we cannot distinguish Slack rows that were marked by this
-- migration from those already in needs_reauth for other reasons.
