-- +goose Up
-- Enable Row-Level Security on all application tables.
--
-- Purpose: Lock down the Supabase PostgREST data API. With RLS enabled and no
-- permissive policies, the anon/authenticated roles cannot read or write any
-- table via PostgREST. The Go backend connects as the postgres superuser, which
-- bypasses RLS entirely — no application code changes are needed.
--
-- ALTER TABLE ... ENABLE ROW LEVEL SECURITY is idempotent: if RLS was already
-- enabled (e.g. via the Supabase dashboard), this is a no-op.

ALTER TABLE profiles ENABLE ROW LEVEL SECURITY;
ALTER TABLE connectors ENABLE ROW LEVEL SECURITY;
ALTER TABLE connector_actions ENABLE ROW LEVEL SECURITY;
ALTER TABLE connector_required_credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE registration_invites ENABLE ROW LEVEL SECURITY;
ALTER TABLE agents ENABLE ROW LEVEL SECURITY;
ALTER TABLE approvals ENABLE ROW LEVEL SECURITY;
ALTER TABLE request_ids ENABLE ROW LEVEL SECURITY;
ALTER TABLE consumed_tokens ENABLE ROW LEVEL SECURITY;
ALTER TABLE credentials ENABLE ROW LEVEL SECURITY;
ALTER TABLE agent_connectors ENABLE ROW LEVEL SECURITY;
ALTER TABLE action_configurations ENABLE ROW LEVEL SECURITY;
ALTER TABLE action_config_templates ENABLE ROW LEVEL SECURITY;
ALTER TABLE standing_approvals ENABLE ROW LEVEL SECURITY;
ALTER TABLE standing_approval_executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE plans ENABLE ROW LEVEL SECURITY;
ALTER TABLE subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE usage_periods ENABLE ROW LEVEL SECURITY;
ALTER TABLE stripe_webhook_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE push_subscriptions ENABLE ROW LEVEL SECURITY;
ALTER TABLE notification_preferences ENABLE ROW LEVEL SECURITY;
ALTER TABLE server_config ENABLE ROW LEVEL SECURITY;

-- +goose Down
ALTER TABLE profiles DISABLE ROW LEVEL SECURITY;
ALTER TABLE connectors DISABLE ROW LEVEL SECURITY;
ALTER TABLE connector_actions DISABLE ROW LEVEL SECURITY;
ALTER TABLE connector_required_credentials DISABLE ROW LEVEL SECURITY;
ALTER TABLE registration_invites DISABLE ROW LEVEL SECURITY;
ALTER TABLE agents DISABLE ROW LEVEL SECURITY;
ALTER TABLE approvals DISABLE ROW LEVEL SECURITY;
ALTER TABLE request_ids DISABLE ROW LEVEL SECURITY;
ALTER TABLE consumed_tokens DISABLE ROW LEVEL SECURITY;
ALTER TABLE credentials DISABLE ROW LEVEL SECURITY;
ALTER TABLE agent_connectors DISABLE ROW LEVEL SECURITY;
ALTER TABLE action_configurations DISABLE ROW LEVEL SECURITY;
ALTER TABLE action_config_templates DISABLE ROW LEVEL SECURITY;
ALTER TABLE standing_approvals DISABLE ROW LEVEL SECURITY;
ALTER TABLE standing_approval_executions DISABLE ROW LEVEL SECURITY;
ALTER TABLE audit_events DISABLE ROW LEVEL SECURITY;
ALTER TABLE plans DISABLE ROW LEVEL SECURITY;
ALTER TABLE subscriptions DISABLE ROW LEVEL SECURITY;
ALTER TABLE usage_periods DISABLE ROW LEVEL SECURITY;
ALTER TABLE stripe_webhook_events DISABLE ROW LEVEL SECURITY;
ALTER TABLE push_subscriptions DISABLE ROW LEVEL SECURITY;
ALTER TABLE notification_preferences DISABLE ROW LEVEL SECURITY;
ALTER TABLE server_config DISABLE ROW LEVEL SECURITY;
