-- +goose Up

-- Create the role with a known local dev password.
-- Production password is set separately via Supabase SQL Editor (see deployment docs).
CREATE ROLE app_backend WITH LOGIN PASSWORD 'app_backend_dev';

GRANT CONNECT ON DATABASE postgres TO app_backend;

GRANT SELECT, INSERT, UPDATE, DELETE ON TABLE
  action_config_templates,
  action_configurations,
  agent_connectors,
  agents,
  approvals,
  audit_events,
  connector_actions,
  connector_required_credentials,
  connectors,
  credentials,
  expo_push_tokens,
  notification_preferences,
  oauth_connections,
  oauth_provider_configs,
  payment_method_transactions,
  payment_methods,
  plans,
  profiles,
  push_subscriptions,
  registration_invites,
  request_ids,
  server_config,
  standing_approval_executions,
  standing_approvals,
  stripe_webhook_events,
  subscriptions,
  usage_periods
TO app_backend;

GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO app_backend;

-- Future tables: default privileges so new migrations don't need manual grants
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO app_backend;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
  GRANT USAGE, SELECT ON SEQUENCES TO app_backend;

-- +goose Down

REVOKE ALL PRIVILEGES ON ALL TABLES IN SCHEMA public FROM app_backend;
REVOKE ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public FROM app_backend;
REVOKE CONNECT ON DATABASE postgres FROM app_backend;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE SELECT, INSERT, UPDATE, DELETE ON TABLES FROM app_backend;
ALTER DEFAULT PRIVILEGES IN SCHEMA public REVOKE USAGE, SELECT ON SEQUENCES FROM app_backend;
DROP ROLE IF EXISTS app_backend;
