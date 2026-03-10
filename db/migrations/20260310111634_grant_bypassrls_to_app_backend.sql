-- +goose Up
-- RLS is enabled on all tables with no permissive policies to block
-- PostgREST/anon access. The app_backend role (non-superuser) needs
-- explicit allow-all policies so it can read/write rows at runtime.
--
-- We use per-table policies instead of BYPASSRLS so the role stays
-- least-privilege — if scoped policies are added later, just drop
-- these blanket policies and replace them.

-- Tables from 20260303220625_enable_rls_all_tables
CREATE POLICY app_backend_all ON profiles FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON connectors FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON connector_actions FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON connector_required_credentials FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON registration_invites FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON agents FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON approvals FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON request_ids FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON credentials FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON agent_connectors FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON action_configurations FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON action_config_templates FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON standing_approvals FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON standing_approval_executions FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON audit_events FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON plans FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON subscriptions FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON usage_periods FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON stripe_webhook_events FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON push_subscriptions FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON notification_preferences FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON server_config FOR ALL TO app_backend USING (true) WITH CHECK (true);

-- Tables from 20260304041308_add_expo_push_tokens
CREATE POLICY app_backend_all ON expo_push_tokens FOR ALL TO app_backend USING (true) WITH CHECK (true);

-- Tables from 20260305045137_enable_rls_oauth_tables
CREATE POLICY app_backend_all ON oauth_connections FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON oauth_provider_configs FOR ALL TO app_backend USING (true) WITH CHECK (true);

-- Tables from 20260306021221_add_payment_methods
CREATE POLICY app_backend_all ON payment_methods FOR ALL TO app_backend USING (true) WITH CHECK (true);
CREATE POLICY app_backend_all ON payment_method_transactions FOR ALL TO app_backend USING (true) WITH CHECK (true);

-- +goose Down
DROP POLICY IF EXISTS app_backend_all ON profiles;
DROP POLICY IF EXISTS app_backend_all ON connectors;
DROP POLICY IF EXISTS app_backend_all ON connector_actions;
DROP POLICY IF EXISTS app_backend_all ON connector_required_credentials;
DROP POLICY IF EXISTS app_backend_all ON registration_invites;
DROP POLICY IF EXISTS app_backend_all ON agents;
DROP POLICY IF EXISTS app_backend_all ON approvals;
DROP POLICY IF EXISTS app_backend_all ON request_ids;
DROP POLICY IF EXISTS app_backend_all ON credentials;
DROP POLICY IF EXISTS app_backend_all ON agent_connectors;
DROP POLICY IF EXISTS app_backend_all ON action_configurations;
DROP POLICY IF EXISTS app_backend_all ON action_config_templates;
DROP POLICY IF EXISTS app_backend_all ON standing_approvals;
DROP POLICY IF EXISTS app_backend_all ON standing_approval_executions;
DROP POLICY IF EXISTS app_backend_all ON audit_events;
DROP POLICY IF EXISTS app_backend_all ON plans;
DROP POLICY IF EXISTS app_backend_all ON subscriptions;
DROP POLICY IF EXISTS app_backend_all ON usage_periods;
DROP POLICY IF EXISTS app_backend_all ON stripe_webhook_events;
DROP POLICY IF EXISTS app_backend_all ON push_subscriptions;
DROP POLICY IF EXISTS app_backend_all ON notification_preferences;
DROP POLICY IF EXISTS app_backend_all ON server_config;
DROP POLICY IF EXISTS app_backend_all ON expo_push_tokens;
DROP POLICY IF EXISTS app_backend_all ON oauth_connections;
DROP POLICY IF EXISTS app_backend_all ON oauth_provider_configs;
DROP POLICY IF EXISTS app_backend_all ON payment_methods;
DROP POLICY IF EXISTS app_backend_all ON payment_method_transactions;
