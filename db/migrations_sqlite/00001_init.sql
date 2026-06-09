-- +goose Up
-- +goose StatementBegin

-- Consolidated initial schema for SQLite. This replaces 115 Postgres migrations
-- in db/migrations/ as part of the Supabase → SQLite migration. Translation
-- rules:
--   uuid                       -> TEXT (Go generates via uuid.NewString())
--   timestamp with time zone   -> TEXT (ISO-8601 UTC, e.g. 2026-05-17T12:34:56.789Z)
--   jsonb                      -> TEXT (validated where needed via json_valid())
--   bigint identity / serial   -> INTEGER PRIMARY KEY AUTOINCREMENT
--   bytea                      -> BLOB
--   text[]                     -> TEXT (JSON-encoded array)
--   DEFAULT now()              -> DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
--   DEFAULT gen_random_uuid()  -> no DEFAULT; caller provides
--   char_length(x) <= N        -> length(x) <= N
--   pg_column_size(x) <= N     -> length(x) <= N  (byte-length approximation)
--   x = ANY (ARRAY['a','b'])   -> x IN ('a','b')
--   jsonb_typeof(x) = 'object' -> json_type(x) = 'object'
--   GIN indexes                -> dropped (Go-side filtering for jsonb @> queries)
--   RLS policies / app_backend -> dropped (Go runs as a single trusted role)
--   pg_cron schedules          -> dropped (Go background jobs in main.go)
--   auth.users                 -> dropped (users table below is our own)
--   PL/pgSQL functions         -> dropped (logic moved to Go)
--   regex CHECKs               -> dropped (validated in Go)
--
-- Triggers are reimplemented in SQLite syntax where they enforce data integrity
-- the application relies on (default connector instance, mixed-auth check).
--
-- Foreign keys require: PRAGMA foreign_keys = ON; (set on every connection
-- by the SQLite driver wrapper).


-- ============================================================================
-- AUTH / IDENTITY
-- ============================================================================

-- Application users. Replaces Supabase auth.users. Password is argon2id-hashed
-- by the auth package; created via /api/auth/signup or cmd/create-user.
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT NOT NULL UNIQUE COLLATE NOCASE,
    password_hash TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CONSTRAINT users_email_format CHECK (length(email) <= 254 AND email LIKE '%_@_%._%')
);

-- Refresh-token sessions. Access JWTs are stateless (verified via JWT_SIGNING_SECRET);
-- refresh tokens are opaque random bytes whose sha256 hash is stored here.
CREATE TABLE auth_sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    refresh_token_hash TEXT NOT NULL UNIQUE,
    expires_at TEXT NOT NULL,
    revoked_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE INDEX idx_auth_sessions_user ON auth_sessions(user_id);
CREATE INDEX idx_auth_sessions_expires ON auth_sessions(expires_at);


-- ============================================================================
-- VAULT (replaces Supabase Vault extension)
-- ============================================================================

-- Encrypted secret storage. Ciphertext is AES-256-GCM with a 12-byte nonce;
-- the master key comes from SECRET_ENCRYPTION_KEY env var (32 random bytes,
-- base64-encoded). See vault/sqlite.go.
CREATE TABLE vault_secrets (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    nonce BLOB NOT NULL,
    ciphertext BLOB NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);


-- ============================================================================
-- CORE DOMAIN TABLES (translated from Postgres dump)
-- ============================================================================

CREATE TABLE profiles (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    email TEXT,
    phone TEXT,
    marketing_opt_in INTEGER NOT NULL DEFAULT 0,
    stripe_customer_id TEXT,
    CONSTRAINT profiles_username_check CHECK (length(username) <= 255),
    CONSTRAINT profiles_email_format CHECK (
        email IS NULL OR (
            email NOT LIKE '% %'
            AND (length(email) - length(replace(email, '@', ''))) = 1
            AND instr(email, '.') > instr(email, '@')
        )
    ),
    CONSTRAINT profiles_phone_e164 CHECK (
        phone IS NULL OR (
            substr(phone, 1, 1) = '+'
            AND length(phone) BETWEEN 2 AND 16
            AND substr(phone, 2, 1) BETWEEN '1' AND '9'
            AND replace(replace(replace(replace(replace(replace(replace(replace(replace(replace(substr(phone, 2), '0', ''), '1', ''), '2', ''), '3', ''), '4', ''), '5', ''), '6', ''), '7', ''), '8', ''), '9', '') = ''
        )
    ),
    FOREIGN KEY (id) REFERENCES users(id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX profiles_username_key ON profiles(username);
CREATE UNIQUE INDEX idx_profiles_stripe_customer_id ON profiles(stripe_customer_id) WHERE stripe_customer_id IS NOT NULL;

CREATE TABLE connectors (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    logo_svg TEXT,
    status TEXT NOT NULL DEFAULT 'untested',
    CONSTRAINT connectors_description_check CHECK (length(description) <= 4096),
    CONSTRAINT connectors_id_check CHECK (length(id) <= 255),
    CONSTRAINT connectors_name_check CHECK (length(name) <= 255),
    CONSTRAINT connectors_status_check CHECK (status IN ('tested', 'early_preview', 'untested'))
);

CREATE TABLE connector_actions (
    connector_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    risk_level TEXT,
    parameters_schema TEXT,
    requires_payment_method INTEGER NOT NULL DEFAULT 0,
    display_template TEXT,
    preview TEXT,
    operation_type TEXT NOT NULL DEFAULT 'write',
    PRIMARY KEY (connector_id, action_type),
    CONSTRAINT connector_actions_action_type_check CHECK (length(action_type) <= 255),
    CONSTRAINT connector_actions_connector_id_check CHECK (length(connector_id) <= 255),
    CONSTRAINT connector_actions_description_check CHECK (length(description) <= 4096),
    CONSTRAINT connector_actions_name_check CHECK (length(name) <= 255),
    CONSTRAINT connector_actions_operation_type_check CHECK (operation_type IN ('read', 'write', 'edit', 'delete')),
    CONSTRAINT connector_actions_parameters_schema_check CHECK (length(parameters_schema) <= 65536),
    CONSTRAINT connector_actions_risk_level_check CHECK (risk_level IS NULL OR risk_level IN ('low', 'medium', 'high')),
    FOREIGN KEY (connector_id) REFERENCES connectors(id) ON DELETE CASCADE
);

CREATE TABLE connector_required_credentials (
    connector_id TEXT NOT NULL,
    service TEXT NOT NULL,
    auth_type TEXT NOT NULL,
    instructions_url TEXT,
    oauth_provider TEXT,
    oauth_scopes TEXT DEFAULT '[]',
    credential_fields TEXT NOT NULL DEFAULT '[]',
    auth_option_group TEXT,
    PRIMARY KEY (connector_id, service, auth_type),
    CONSTRAINT chk_oauth_provider_required CHECK (
        (auth_type = 'oauth2' AND oauth_provider IS NOT NULL)
        OR (auth_type <> 'oauth2' AND oauth_provider IS NULL)
    ),
    CONSTRAINT connector_required_credentials_auth_option_group_check CHECK (length(auth_option_group) <= 255),
    CONSTRAINT connector_required_credentials_auth_type_check CHECK (auth_type IN ('api_key', 'basic', 'custom', 'oauth2')),
    CONSTRAINT connector_required_credentials_connector_id_check CHECK (length(connector_id) <= 255),
    CONSTRAINT connector_required_credentials_instructions_url_check CHECK (length(instructions_url) <= 2048),
    CONSTRAINT connector_required_credentials_oauth_provider_check CHECK (length(oauth_provider) <= 255),
    CONSTRAINT connector_required_credentials_service_check CHECK (length(service) <= 255),
    FOREIGN KEY (connector_id) REFERENCES connectors(id) ON DELETE CASCADE
);

CREATE TABLE agents (
    agent_id INTEGER PRIMARY KEY AUTOINCREMENT,
    public_key TEXT NOT NULL,
    approver_id TEXT NOT NULL,
    status TEXT NOT NULL,
    metadata TEXT,
    verification_attempts INTEGER NOT NULL DEFAULT 0,
    registration_ttl INTEGER,
    expires_at TEXT,
    registered_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    deactivated_at TEXT,
    last_active_at TEXT,
    confirmation_code TEXT,
    CONSTRAINT agents_confirmation_code_check CHECK (length(confirmation_code) <= 11),
    CONSTRAINT agents_metadata_check CHECK (length(metadata) <= 65536),
    CONSTRAINT agents_public_key_check CHECK (length(public_key) <= 1024),
    CONSTRAINT agents_registration_ttl_check CHECK (registration_ttl IS NULL OR (registration_ttl >= 60 AND registration_ttl <= 86400)),
    CONSTRAINT agents_status_check CHECK (status IN ('pending', 'registered', 'deactivated')),
    CONSTRAINT agents_verification_attempts_check CHECK (verification_attempts >= 0),
    FOREIGN KEY (approver_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_agents_approver_created ON agents(approver_id, created_at DESC, agent_id DESC);
CREATE INDEX idx_agents_approver_status ON agents(approver_id, status);

CREATE TABLE agent_connectors (
    agent_id INTEGER NOT NULL,
    approver_id TEXT NOT NULL,
    connector_id TEXT NOT NULL,
    connector_instance_id TEXT NOT NULL,
    enabled_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    is_default INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (agent_id, approver_id, connector_id, connector_instance_id),
    CONSTRAINT agent_connectors_connector_id_check CHECK (length(connector_id) <= 255),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (connector_id) REFERENCES connectors(id) ON DELETE CASCADE
);
CREATE INDEX idx_agent_connectors_connector ON agent_connectors(connector_id);
CREATE UNIQUE INDEX uq_agent_connectors_default_per_pair ON agent_connectors(agent_id, connector_id) WHERE is_default = 1;

CREATE TABLE agent_connector_credentials (
    id TEXT PRIMARY KEY,
    agent_id INTEGER NOT NULL,
    connector_id TEXT NOT NULL,
    approver_id TEXT NOT NULL,
    credential_id TEXT,
    oauth_connection_id TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    connector_instance_id TEXT NOT NULL,
    CONSTRAINT agent_connector_credentials_check CHECK (
        (credential_id IS NOT NULL AND oauth_connection_id IS NULL)
        OR (credential_id IS NULL AND oauth_connection_id IS NOT NULL)
    ),
    CONSTRAINT agent_connector_credentials_connector_id_check CHECK (length(connector_id) <= 255),
    CONSTRAINT agent_connector_credentials_id_check CHECK (length(id) <= 255),
    FOREIGN KEY (agent_id, approver_id, connector_id, connector_instance_id)
        REFERENCES agent_connectors(agent_id, approver_id, connector_id, connector_instance_id)
        ON DELETE CASCADE
    -- credential_id and oauth_connection_id FKs declared after those tables exist
);
CREATE INDEX idx_agent_connector_credentials_cred ON agent_connector_credentials(credential_id);
CREATE INDEX idx_agent_connector_credentials_oauth ON agent_connector_credentials(oauth_connection_id);
CREATE UNIQUE INDEX idx_agent_connector_credentials_unique ON agent_connector_credentials(agent_id, connector_id, connector_instance_id);

CREATE TABLE credentials (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    service TEXT NOT NULL,
    label TEXT,
    vault_secret_id TEXT NOT NULL,
    connector_state TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CONSTRAINT credentials_id_check CHECK (length(id) <= 255),
    CONSTRAINT credentials_label_check CHECK (length(label) <= 255),
    CONSTRAINT credentials_service_check CHECK (length(service) <= 255),
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_credentials_user_service ON credentials(user_id, service);
-- Postgres uses `UNIQUE NULLS NOT DISTINCT (user_id, service, label)` so that
-- two rows with the same (user, service) and NULL label collide. SQLite treats
-- NULLs as distinct in UNIQUE, so we split into two partial indexes.
CREATE UNIQUE INDEX credentials_user_service_label_notnull
    ON credentials(user_id, service, label) WHERE label IS NOT NULL;
CREATE UNIQUE INDEX credentials_user_service_label_null
    ON credentials(user_id, service) WHERE label IS NULL;

CREATE TABLE oauth_connections (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    provider TEXT NOT NULL,
    access_token_vault_id TEXT NOT NULL,
    refresh_token_vault_id TEXT,
    scopes TEXT NOT NULL DEFAULT '[]',
    token_expiry TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    extra_data TEXT,
    CONSTRAINT oauth_connections_id_check CHECK (length(id) <= 255),
    CONSTRAINT oauth_connections_provider_check CHECK (length(provider) <= 255),
    CONSTRAINT oauth_connections_status_check CHECK (status IN ('active', 'needs_reauth', 'revoked')),
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_oauth_connections_user ON oauth_connections(user_id);
CREATE INDEX idx_oauth_connections_user_provider ON oauth_connections(user_id, provider);
CREATE INDEX idx_oauth_connections_status ON oauth_connections(user_id, status);

CREATE TABLE approvals (
    approval_id TEXT PRIMARY KEY,
    approver_id TEXT NOT NULL,
    action TEXT NOT NULL,
    context TEXT NOT NULL,
    status TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    approved_at TEXT,
    denied_at TEXT,
    cancelled_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    agent_id INTEGER NOT NULL,
    execution_status TEXT,
    execution_result TEXT,
    executed_at TEXT,
    resource_details TEXT,
    denial_reason TEXT,
    action_fingerprint TEXT,
    CONSTRAINT approvals_action_check CHECK (length(action) <= 65536),
    CONSTRAINT approvals_approval_id_check CHECK (length(approval_id) <= 255),
    CONSTRAINT approvals_context_check CHECK (length(context) <= 262144),
    CONSTRAINT approvals_denial_reason_check CHECK (denial_reason IS NULL OR length(denial_reason) <= 500),
    CONSTRAINT approvals_action_fingerprint_check CHECK (action_fingerprint IS NULL OR length(action_fingerprint) <= 64),
    CONSTRAINT approvals_execution_status_check CHECK (execution_status IS NULL OR execution_status IN ('pending', 'success', 'error')),
    CONSTRAINT approvals_status_check CHECK (status IN ('pending', 'approved', 'denied', 'cancelled')),
    CONSTRAINT chk_execution_columns_consistent CHECK (
        (execution_status IS NULL AND executed_at IS NULL)
        OR (execution_status IS NOT NULL AND executed_at IS NOT NULL)
    ),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (approver_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_approvals_agent_created ON approvals(agent_id, created_at);
CREATE INDEX idx_approvals_agent_status ON approvals(agent_id, status);
CREATE INDEX idx_approvals_approver_created ON approvals(approver_id, created_at DESC, approval_id DESC);
CREATE INDEX idx_approvals_expires_at ON approvals(expires_at);
CREATE INDEX idx_approvals_denial_dedup ON approvals(agent_id, approver_id, action_fingerprint, denied_at DESC) WHERE status = 'denied';

CREATE TABLE action_configurations (
    id TEXT PRIMARY KEY,
    agent_id INTEGER NOT NULL,
    user_id TEXT NOT NULL,
    connector_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    parameters TEXT NOT NULL DEFAULT '{}',
    status TEXT NOT NULL DEFAULT 'active',
    name TEXT NOT NULL,
    description TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CONSTRAINT action_configurations_action_type_check CHECK (length(action_type) <= 255),
    CONSTRAINT action_configurations_connector_id_check CHECK (length(connector_id) <= 255),
    CONSTRAINT action_configurations_description_check CHECK (length(description) <= 4096),
    CONSTRAINT action_configurations_id_check CHECK (length(id) <= 255),
    CONSTRAINT action_configurations_name_check CHECK (length(name) <= 255),
    CONSTRAINT action_configurations_parameters_check CHECK (length(parameters) <= 65536),
    CONSTRAINT action_configurations_status_check CHECK (status IN ('active', 'disabled')),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (connector_id) REFERENCES connectors(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_action_configurations_agent ON action_configurations(agent_id, user_id);
CREATE INDEX idx_action_configurations_connector_action ON action_configurations(connector_id, action_type);
CREATE UNIQUE INDEX idx_action_config_wildcard_unique ON action_configurations(agent_id, connector_id) WHERE action_type = '*';

CREATE TABLE action_config_templates (
    id TEXT PRIMARY KEY,
    connector_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT,
    parameters TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    standing_approval_spec TEXT,
    CONSTRAINT action_config_templates_action_type_check CHECK (length(action_type) <= 255),
    CONSTRAINT action_config_templates_connector_id_check CHECK (length(connector_id) <= 255),
    CONSTRAINT action_config_templates_description_check CHECK (length(description) <= 4096),
    CONSTRAINT action_config_templates_id_check CHECK (length(id) <= 255),
    CONSTRAINT action_config_templates_name_check CHECK (length(name) <= 255),
    CONSTRAINT action_config_templates_parameters_check CHECK (json_type(parameters) = 'object'),
    CONSTRAINT action_config_templates_parameters_check1 CHECK (length(parameters) <= 65536),
    CONSTRAINT action_config_templates_standing_approval_spec_check CHECK (standing_approval_spec IS NULL OR json_type(standing_approval_spec) = 'object'),
    CONSTRAINT action_config_templates_standing_approval_spec_check1 CHECK (standing_approval_spec IS NULL OR length(standing_approval_spec) <= 4096),
    FOREIGN KEY (connector_id) REFERENCES connectors(id) ON DELETE CASCADE,
    FOREIGN KEY (connector_id, action_type) REFERENCES connector_actions(connector_id, action_type) ON DELETE CASCADE
);
CREATE INDEX idx_action_config_templates_connector ON action_config_templates(connector_id);

CREATE TABLE standing_approvals (
    standing_approval_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    action_type TEXT NOT NULL,
    action_version TEXT NOT NULL DEFAULT '1',
    constraints TEXT,
    status TEXT NOT NULL,
    starts_at TEXT NOT NULL,
    expires_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    revoked_at TEXT,
    expired_at TEXT,
    agent_id INTEGER NOT NULL,
    source_action_configuration_id TEXT NOT NULL,
    connector_instance_id TEXT,
    CONSTRAINT standing_approvals_action_type_check CHECK (length(action_type) <= 128),
    CONSTRAINT standing_approvals_action_version_check CHECK (length(action_version) <= 10),
    CONSTRAINT standing_approvals_constraints_check CHECK (length(constraints) <= 65536),
    CONSTRAINT standing_approvals_expires_at_after_starts_at CHECK (expires_at IS NULL OR expires_at >= starts_at),
    CONSTRAINT standing_approvals_standing_approval_id_check CHECK (length(standing_approval_id) <= 255),
    CONSTRAINT standing_approvals_status_check CHECK (status IN ('active', 'expired', 'revoked')),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (source_action_configuration_id) REFERENCES action_configurations(id) ON DELETE RESTRICT,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_standing_approvals_agent_action_status ON standing_approvals(agent_id, action_type, status);
CREATE INDEX idx_standing_approvals_agent_action_status_connector_instance ON standing_approvals(agent_id, action_type, status, connector_instance_id);
CREATE INDEX idx_standing_approvals_agent_status_created ON standing_approvals(agent_id, status, created_at DESC, standing_approval_id DESC);
CREATE INDEX idx_standing_approvals_source_config_active ON standing_approvals(source_action_configuration_id) WHERE status = 'active';
CREATE INDEX idx_standing_approvals_user_active ON standing_approvals(user_id) WHERE status = 'active';
-- GIN index on constraints jsonb_path_ops dropped — SQLite doesn't have GIN.
-- The few JSON containment queries are handled Go-side via filtering.

CREATE TABLE standing_approval_executions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    standing_approval_id TEXT NOT NULL,
    parameters TEXT,
    executed_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    request_id TEXT,
    CONSTRAINT standing_approval_executions_parameters_check CHECK (length(parameters) <= 65536),
    FOREIGN KEY (standing_approval_id) REFERENCES standing_approvals(standing_approval_id) ON DELETE CASCADE
);
CREATE INDEX idx_sa_executions_sa_id ON standing_approval_executions(standing_approval_id);
CREATE UNIQUE INDEX idx_sa_executions_request_id ON standing_approval_executions(standing_approval_id, request_id) WHERE request_id IS NOT NULL;

CREATE TABLE audit_events (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    agent_id INTEGER NOT NULL,
    event_type TEXT NOT NULL,
    outcome TEXT NOT NULL,
    source_id TEXT,
    source_type TEXT,
    agent_meta TEXT,
    action TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    execution_status TEXT,
    execution_error TEXT,
    connector_id TEXT,
    CONSTRAINT audit_events_event_type_check CHECK (event_type IN (
        'approval.requested', 'approval.approved', 'approval.denied', 'approval.cancelled',
        'action.executed', 'standing_approval.executed', 'standing_approval.updated',
        'agent.registered', 'agent.deactivated', 'payment_method.charged'
    )),
    CONSTRAINT audit_events_execution_status_check CHECK (execution_status IS NULL OR execution_status IN ('success', 'failure', 'timeout', 'skipped')),
    CONSTRAINT audit_events_outcome_check CHECK (outcome IN (
        'approved', 'denied', 'cancelled', 'auto_executed', 'registered',
        'deactivated', 'pending', 'expired', 'charged', 'updated'
    )),
    CONSTRAINT audit_events_source_type_check CHECK (source_type IS NULL OR source_type IN (
        'approval', 'standing_approval', 'agent', 'registration_invite', 'payment_method_transaction'
    )),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_audit_events_agent ON audit_events(agent_id, created_at DESC, id DESC);
CREATE INDEX idx_audit_events_connector_created ON audit_events(connector_id, created_at DESC) WHERE connector_id IS NOT NULL;
CREATE INDEX idx_audit_events_source_resolution ON audit_events(source_id, user_id, event_type) WHERE source_id IS NOT NULL;
CREATE INDEX idx_audit_events_user_created ON audit_events(user_id, created_at DESC, id DESC);

CREATE TABLE consumed_signatures (
    signature_hash BLOB PRIMARY KEY,
    agent_id INTEGER NOT NULL,
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);
CREATE INDEX idx_consumed_signatures_expires_at ON consumed_signatures(expires_at);

CREATE TABLE registration_invites (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    invite_code_hash TEXT NOT NULL UNIQUE,
    status TEXT NOT NULL,
    verification_attempts INTEGER NOT NULL DEFAULT 0,
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    CONSTRAINT registration_invites_id_check CHECK (length(id) <= 255),
    CONSTRAINT registration_invites_invite_code_hash_check CHECK (length(invite_code_hash) <= 128),
    CONSTRAINT registration_invites_status_check CHECK (status IN ('active', 'consumed', 'expired')),
    CONSTRAINT registration_invites_verification_attempts_check CHECK (verification_attempts >= 0),
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_registration_invites_user_status ON registration_invites(user_id, status);

CREATE TABLE request_ids (
    request_id TEXT PRIMARY KEY,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    approver_id TEXT NOT NULL,
    agent_id INTEGER NOT NULL,
    CONSTRAINT request_ids_request_id_check CHECK (length(request_id) <= 255),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE
);
CREATE INDEX idx_request_ids_created_at ON request_ids(created_at);

CREATE TABLE server_config (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
);

CREATE TABLE push_subscriptions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    endpoint TEXT,
    p256dh TEXT,
    auth TEXT,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    channel TEXT NOT NULL DEFAULT 'web-push',
    expo_token TEXT,
    CONSTRAINT push_subscriptions_channel_check CHECK (channel IN ('web-push', 'mobile-push')),
    CONSTRAINT push_subscriptions_mobile_push_fields CHECK (channel <> 'mobile-push' OR expo_token IS NOT NULL),
    CONSTRAINT push_subscriptions_web_push_fields CHECK (
        channel <> 'web-push' OR (endpoint IS NOT NULL AND p256dh IS NOT NULL AND auth IS NOT NULL)
    ),
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_push_subscriptions_user_id ON push_subscriptions(user_id);
CREATE UNIQUE INDEX idx_push_subscriptions_user_endpoint ON push_subscriptions(user_id, endpoint) WHERE endpoint IS NOT NULL;
CREATE UNIQUE INDEX idx_push_subscriptions_user_expo_token ON push_subscriptions(user_id, expo_token) WHERE expo_token IS NOT NULL;

CREATE TABLE expo_push_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    token TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    UNIQUE (user_id, token),
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_expo_push_tokens_user_id ON expo_push_tokens(user_id);

CREATE TABLE notification_preferences (
    user_id TEXT NOT NULL,
    channel TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (user_id, channel),
    CONSTRAINT notification_preferences_channel_check CHECK (channel IN ('email', 'web-push', 'sms', 'mobile-push')),
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);

CREATE TABLE notification_type_preferences (
    user_id TEXT NOT NULL,
    notification_type TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    PRIMARY KEY (user_id, notification_type),
    CONSTRAINT notification_type_preferences_notification_type_check CHECK (notification_type = 'standing_execution'),
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON DELETE CASCADE
);

CREATE TABLE payment_methods (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    stripe_payment_method_id TEXT NOT NULL UNIQUE,
    label TEXT NOT NULL DEFAULT '',
    brand TEXT NOT NULL DEFAULT '',
    last4 TEXT NOT NULL DEFAULT '',
    exp_month INTEGER NOT NULL DEFAULT 0,
    exp_year INTEGER NOT NULL DEFAULT 0,
    billing_address_line1 TEXT NOT NULL DEFAULT '',
    billing_address_line2 TEXT NOT NULL DEFAULT '',
    billing_address_city TEXT NOT NULL DEFAULT '',
    billing_address_state TEXT NOT NULL DEFAULT '',
    billing_address_postal TEXT NOT NULL DEFAULT '',
    billing_address_country TEXT NOT NULL DEFAULT '',
    is_default INTEGER NOT NULL DEFAULT 0,
    per_transaction_limit INTEGER,
    monthly_limit INTEGER,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    expiration_alert_sent_at TEXT,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_payment_methods_user_id ON payment_methods(user_id);
CREATE UNIQUE INDEX idx_payment_methods_user_default ON payment_methods(user_id) WHERE is_default = 1;

CREATE TABLE payment_method_transactions (
    id TEXT PRIMARY KEY,
    payment_method_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    connector_id TEXT NOT NULL DEFAULT '',
    action_type TEXT NOT NULL DEFAULT '',
    amount_cents INTEGER NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id) ON DELETE CASCADE,
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);
CREATE INDEX idx_pmt_payment_method_id ON payment_method_transactions(payment_method_id);
CREATE INDEX idx_pmt_user_created ON payment_method_transactions(user_id, created_at);

CREATE TABLE agent_payment_methods (
    agent_id INTEGER PRIMARY KEY,
    payment_method_id TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    FOREIGN KEY (agent_id) REFERENCES agents(agent_id) ON DELETE CASCADE,
    FOREIGN KEY (payment_method_id) REFERENCES payment_methods(id) ON DELETE CASCADE
);
CREATE INDEX idx_agent_payment_methods_pm ON agent_payment_methods(payment_method_id);

CREATE TABLE usage_periods (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id TEXT NOT NULL,
    period_start TEXT NOT NULL,
    period_end TEXT NOT NULL,
    request_count INTEGER NOT NULL DEFAULT 0,
    sms_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ', 'now')),
    breakdown TEXT NOT NULL DEFAULT '{}',
    UNIQUE (user_id, period_start),
    CONSTRAINT chk_usage_periods_valid_range CHECK (period_end > period_start),
    FOREIGN KEY (user_id) REFERENCES profiles(id) ON UPDATE CASCADE ON DELETE CASCADE
);

-- +goose StatementEnd

-- ============================================================================
-- TRIGGERS (reimplemented from PL/pgSQL functions)
-- ============================================================================

-- +goose StatementBegin
-- First row inserted for (agent_id, connector_id) becomes the default instance.
-- Reimplements trg_agent_connectors_before_insert.
CREATE TRIGGER trg_agent_connectors_default_first
AFTER INSERT ON agent_connectors
FOR EACH ROW
WHEN NEW.is_default = 0
  AND NOT EXISTS (
    SELECT 1 FROM agent_connectors
    WHERE agent_id = NEW.agent_id
      AND connector_id = NEW.connector_id
      AND is_default = 1
  )
BEGIN
    UPDATE agent_connectors
       SET is_default = 1
     WHERE agent_id = NEW.agent_id
       AND approver_id = NEW.approver_id
       AND connector_id = NEW.connector_id
       AND connector_instance_id = NEW.connector_instance_id;
END;
-- +goose StatementEnd

-- +goose StatementBegin
-- Prevent mixing oauth2 and non-oauth2 auth types for the same connector outside
-- of an auth_option_group. Reimplements check_no_mixed_auth_types.
CREATE TRIGGER trg_no_mixed_auth_types_insert
BEFORE INSERT ON connector_required_credentials
FOR EACH ROW
WHEN EXISTS (
    SELECT 1 FROM connector_required_credentials
    WHERE connector_id = NEW.connector_id
      AND ((auth_type = 'oauth2') <> (NEW.auth_type = 'oauth2'))
      AND NOT (
          NEW.auth_option_group IS NOT NULL
          AND auth_option_group IS NEW.auth_option_group
      )
)
BEGIN
    SELECT RAISE(ABORT, 'connector cannot mix oauth2 and non-oauth2 auth types outside of an auth_option_group');
END;
-- +goose StatementEnd

-- +goose StatementBegin
CREATE TRIGGER trg_no_mixed_auth_types_update
BEFORE UPDATE ON connector_required_credentials
FOR EACH ROW
WHEN EXISTS (
    SELECT 1 FROM connector_required_credentials
    WHERE connector_id = NEW.connector_id
      AND NOT (connector_id = OLD.connector_id AND service = OLD.service AND auth_type = OLD.auth_type)
      AND ((auth_type = 'oauth2') <> (NEW.auth_type = 'oauth2'))
      AND NOT (
          NEW.auth_option_group IS NOT NULL
          AND auth_option_group IS NEW.auth_option_group
      )
)
BEGIN
    SELECT RAISE(ABORT, 'connector cannot mix oauth2 and non-oauth2 auth types outside of an auth_option_group');
END;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS usage_periods;
DROP TABLE IF EXISTS agent_payment_methods;
DROP TABLE IF EXISTS payment_method_transactions;
DROP TABLE IF EXISTS payment_methods;
DROP TABLE IF EXISTS notification_type_preferences;
DROP TABLE IF EXISTS notification_preferences;
DROP TABLE IF EXISTS expo_push_tokens;
DROP TABLE IF EXISTS push_subscriptions;
DROP TABLE IF EXISTS server_config;
DROP TABLE IF EXISTS request_ids;
DROP TABLE IF EXISTS registration_invites;
DROP TABLE IF EXISTS consumed_signatures;
DROP TABLE IF EXISTS audit_events;
DROP TABLE IF EXISTS standing_approval_executions;
DROP TABLE IF EXISTS standing_approvals;
DROP TABLE IF EXISTS action_config_templates;
DROP TABLE IF EXISTS action_configurations;
DROP TABLE IF EXISTS approvals;
DROP TABLE IF EXISTS oauth_connections;
DROP TABLE IF EXISTS credentials;
DROP TABLE IF EXISTS agent_connector_credentials;
DROP TABLE IF EXISTS agent_connectors;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS connector_required_credentials;
DROP TABLE IF EXISTS connector_actions;
DROP TABLE IF EXISTS connectors;
DROP INDEX IF EXISTS idx_profiles_stripe_customer_id;
DROP INDEX IF EXISTS profiles_username_key;
DROP TABLE IF EXISTS profiles;
DROP TABLE IF EXISTS vault_secrets;
DROP TABLE IF EXISTS auth_sessions;
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
