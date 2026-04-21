package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/oauth"
	"github.com/supersuit-tech/permission-slip/vault"
)

// ── test helpers ────────────────────────────────────────────────────────────

// oauthExecFixture holds the common setup for OAuth execution tests.
type oauthExecFixture struct {
	TX       db.DBTX
	Deps     *Deps
	UserID   string
	AgentID  int64
	Vault    *vault.MockVaultStore
	ConnReg  *connectors.Registry
	OAuthReg *oauth.Registry
}

// oauthExecOpts configures setupOAuthExecutionTest.
type oauthExecOpts struct {
	// ConnectorID defaults to "testgoogle".
	ConnectorID string
	// ActionType defaults to "<connectorID>.send_email".
	ActionType string
	// Provider defaults to "google".
	Provider string
	// Connection, if set, creates an oauth_connection with these options.
	// Leave nil to test the "no connection" case.
	Connection *testhelper.OAuthConnectionOpts
	// NoVault disables the vault (deps.Vault = nil).
	NoVault bool
	// NoOAuthRegistry disables the OAuth registry (deps.OAuthProviders = nil).
	NoOAuthRegistry bool
	// OnExec is called with the credentials during connector execution.
	// If nil, a no-op stub connector is used.
	OnExec func(connectors.Credentials)
}

// setupOAuthExecutionTest creates the full fixture for testing executeConnectorAction
// with OAuth connectors. It reduces the ~25 lines of boilerplate per test to a single call.
func setupOAuthExecutionTest(t *testing.T, opts oauthExecOpts) oauthExecFixture {
	t.Helper()

	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := opts.ConnectorID
	if connID == "" {
		connID = "testgoogle"
	}
	provider := opts.Provider
	if provider == "" {
		provider = "google"
	}
	actionType := opts.ActionType
	if actionType == "" {
		actionType = connID + ".send_email"
	}

	// DB: connector + action + OAuth credential requirement.
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, actionType, "Test Action")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, provider, provider, nil)

	// Agent + agent_connector (required for credential resolution).
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	// Vault.
	v := vault.NewMockVaultStore()

	// OAuth connection (optional) + credential binding.
	if opts.Connection != nil {
		connOAuthID := testhelper.GenerateID(t, "oconn_")
		// Default scopes to empty array to satisfy NOT NULL constraint.
		if opts.Connection.Scopes == nil {
			opts.Connection.Scopes = []string{}
		}
		testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, provider, *opts.Connection)

		// Create an explicit credential binding to this OAuth connection.
		conn, connErr := db.GetOAuthConnectionByProvider(t.Context(), tx, uid, provider)
		if connErr != nil {
			t.Fatalf("get oauth connection: %v", connErr)
		}
		bindingID := testhelper.GenerateID(t, "accr_")
		_, bindErr := db.UpsertAgentConnectorCredential(t.Context(), tx, db.UpsertAgentConnectorCredentialParams{
			ID: bindingID, AgentID: agentID, ConnectorID: connID,
			ApproverID: uid, OAuthConnectionID: &conn.ID,
		})
		if bindErr != nil {
			t.Fatalf("upsert binding: %v", bindErr)
		}
	}

	// Connector registry.
	reg := connectors.NewRegistry()
	if opts.OnExec != nil {
		reg.Register(&credCapturingConnector{
			id:      connID,
			actions: []string{actionType},
			onExec:  opts.OnExec,
		})
	} else {
		reg.Register(newTestStubConnector(connID, actionType))
	}

	// OAuth provider registry.
	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID:           provider,
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Source:       oauth.SourceBuiltIn,
	})

	deps := &Deps{
		DB:             tx,
		Vault:          v,
		Connectors:     reg,
		OAuthProviders: oauthReg,
	}
	if opts.NoVault {
		deps.Vault = nil
	}
	if opts.NoOAuthRegistry {
		deps.OAuthProviders = nil
	}

	return oauthExecFixture{
		TX:       tx,
		Deps:     deps,
		UserID:   uid,
		AgentID:  agentID,
		Vault:    v,
		ConnReg:  reg,
		OAuthReg: oauthReg,
	}
}

// credCapturingConnector is a test connector that captures the credentials
// passed during execution.
type credCapturingConnector struct {
	id      string
	actions []string
	onExec  func(connectors.Credentials)
}

func (c *credCapturingConnector) ID() string { return c.id }

func (c *credCapturingConnector) Actions() map[string]connectors.Action {
	m := make(map[string]connectors.Action, len(c.actions))
	for _, at := range c.actions {
		m[at] = &credCapturingAction{onExec: c.onExec}
	}
	return m
}

func (c *credCapturingConnector) ValidateCredentials(_ context.Context, _ connectors.Credentials) error {
	return nil
}

type credCapturingAction struct {
	onExec func(connectors.Credentials)
}

func (a *credCapturingAction) Execute(_ context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	if a.onExec != nil {
		a.onExec(req.Credentials)
	}
	return &connectors.ActionResult{}, nil
}

// newTestStubConnector creates a minimal stub connector for tests.
func newTestStubConnector(id string, actionTypes ...string) *testStubConnector {
	return &testStubConnector{id: id, actionTypes: actionTypes}
}

type testStubConnector struct {
	id          string
	actionTypes []string
}

func (c *testStubConnector) ID() string { return c.id }

func (c *testStubConnector) Actions() map[string]connectors.Action {
	m := make(map[string]connectors.Action, len(c.actionTypes))
	for _, at := range c.actionTypes {
		m[at] = &testStubAction{}
	}
	return m
}

func (c *testStubConnector) ValidateCredentials(_ context.Context, _ connectors.Credentials) error {
	return nil
}

type testStubAction struct{}

func (a *testStubAction) Execute(_ context.Context, _ connectors.ActionRequest) (*connectors.ActionResult, error) {
	return &connectors.ActionResult{}, nil
}

// ── executeConnectorAction: OAuth path ──────────────────────────────────────

func TestExecuteConnectorAction_OAuthPath_Success(t *testing.T) {
	t.Parallel()

	var capturedCreds connectors.Credentials
	f := setupOAuthExecutionTest(t, oauthExecOpts{
		OnExec: func(creds connectors.Credentials) { capturedCreds = creds },
		Connection: &testhelper.OAuthConnectionOpts{
			Scopes: []string{"https://www.googleapis.com/auth/gmail.send"},
			Status: "active",
		},
	})

	// Store access token in vault and update the connection to reference it.
	accessVaultID, err := f.Vault.CreateSecret(t.Context(), f.TX, "access", []byte("valid-access-token"))
	if err != nil {
		t.Fatalf("vault create: %v", err)
	}
	futureExpiry := time.Now().Add(1 * time.Hour)
	conn, err := db.GetOAuthConnectionByProvider(t.Context(), f.TX, f.UserID, "google")
	if err != nil || conn == nil {
		t.Fatalf("get connection: %v", err)
	}
	if err := db.UpdateOAuthConnectionTokens(t.Context(), f.TX, conn.ID, f.UserID, accessVaultID, nil, &futureExpiry); err != nil {
		t.Fatalf("update tokens: %v", err)
	}

	result, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	tok, ok := capturedCreds.Get("access_token")
	if !ok {
		t.Fatal("expected access_token in credentials")
	}
	if tok != "valid-access-token" {
		t.Errorf("expected access_token %q, got %q", "valid-access-token", tok)
	}
}

func TestExecuteConnectorAction_OAuthPath_NoConnection_NoBinding(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{
		// No Connection — no binding will be created either.
	})

	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil, "")
	if err == nil {
		t.Fatal("expected error when no credential binding exists")
	}
	// Without a binding, the error should tell the user to assign a credential.
	if !strings.Contains(err.Error(), "no credential assigned") {
		t.Errorf("expected 'no credential assigned' error, got: %v", err)
	}
}

// TestExecuteConnectorAction_OAuth_NoBinding_ReturnsError verifies that when a
// connector requires OAuth but no credential binding exists, execution fails
// with a clear error instead of auto-resolving.
// (Previously tested with dual oauth2+api_key auth, but mixed auth types are
// now prevented at the schema level per issue #803.)
func TestExecuteConnectorAction_OAuth_NoBinding_ReturnsError(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{
		ConnectorID: "testnotion",
		ActionType:  "testnotion.search",
		Provider:    "notion",
		// No OAuth connection — no binding will be created.
	})

	// Without a binding, execution should fail even though the connector exists.
	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testnotion.search", json.RawMessage(`{}`), nil, "")
	if err == nil {
		t.Fatal("expected error when no credential binding exists")
	}
	if !strings.Contains(err.Error(), "no credential assigned") {
		t.Errorf("expected 'no credential assigned' error, got: %v", err)
	}
}

// TestExecuteConnectorAction_OAuth_NeedsReauth verifies that when a connector
// requires OAuth and the connection has status needs_reauth, execution returns
// an OAuthRefreshError instead of proceeding.
// (Previously tested with dual oauth2+api_key auth to verify no fallback, but
// mixed auth types are now prevented at the schema level per issue #803.)
func TestExecuteConnectorAction_OAuth_NeedsReauth(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{
		ConnectorID: "testnotion",
		ActionType:  "testnotion.search",
		Provider:    "notion",
		Connection:  &testhelper.OAuthConnectionOpts{Status: "needs_reauth"},
	})

	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testnotion.search", json.RawMessage(`{}`), nil, "")
	if err == nil {
		t.Fatal("expected error for needs_reauth status")
	}
	if !connectors.IsOAuthRefreshError(err) {
		t.Errorf("expected OAuthRefreshError, got %T: %v", err, err)
	}
}

func TestExecuteConnectorAction_OAuthPath_NeedsReauth(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{
		Connection: &testhelper.OAuthConnectionOpts{Status: "needs_reauth"},
	})

	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil, "")
	if err == nil {
		t.Fatal("expected error for needs_reauth connection")
	}
	if !connectors.IsOAuthRefreshError(err) {
		t.Errorf("expected OAuthRefreshError, got %T: %v", err, err)
	}
}

func TestExecuteConnectorAction_OAuthPath_RevokedConnection(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{
		ActionType: "testgoogle.list_emails",
		Connection: &testhelper.OAuthConnectionOpts{Status: "revoked"},
	})

	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.list_emails", json.RawMessage(`{}`), nil, "")
	if err == nil {
		t.Fatal("expected error for revoked connection")
	}
	if !connectors.IsOAuthRefreshError(err) {
		t.Errorf("expected OAuthRefreshError, got %T: %v", err, err)
	}
}

func TestExecuteConnectorAction_OAuthPath_NoVault(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{NoVault: true})

	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil, "")
	if err == nil {
		t.Fatal("expected error when vault is nil")
	}
}

func TestExecuteConnectorAction_OAuthPath_NoOAuthRegistry(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{NoOAuthRegistry: true})

	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil, "")
	if err == nil {
		t.Fatal("expected error when OAuth registry is nil")
	}
}

func TestExecuteConnectorAction_OAuthPath_ExpiredTokenNoRefreshToken(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{
		Connection: &testhelper.OAuthConnectionOpts{Status: "active"},
	})

	// Store an expired access token (no refresh token).
	accessVaultID, _ := f.Vault.CreateSecret(t.Context(), f.TX, "access", []byte("expired-token"))
	pastExpiry := time.Now().Add(-10 * time.Minute)
	conn, _ := db.GetOAuthConnectionByProvider(t.Context(), f.TX, f.UserID, "google")
	_ = db.UpdateOAuthConnectionTokens(t.Context(), f.TX, conn.ID, f.UserID, accessVaultID, nil, &pastExpiry)

	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil, "")
	if err == nil {
		t.Fatal("expected error for expired token without refresh token")
	}
	if !connectors.IsOAuthRefreshError(err) {
		t.Errorf("expected OAuthRefreshError, got %T: %v", err, err)
	}

	// Verify connection status was updated to needs_reauth.
	updated, err := db.GetOAuthConnectionByProvider(t.Context(), f.TX, f.UserID, "google")
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	if updated.Status != db.OAuthStatusNeedsReauth {
		t.Errorf("expected status %q, got %q", db.OAuthStatusNeedsReauth, updated.Status)
	}
}

func TestExecuteConnectorAction_OAuthPath_NonExpiredTokenSkipsRefresh(t *testing.T) {
	t.Parallel()

	var capturedCreds connectors.Credentials
	f := setupOAuthExecutionTest(t, oauthExecOpts{
		OnExec:     func(creds connectors.Credentials) { capturedCreds = creds },
		Connection: &testhelper.OAuthConnectionOpts{Status: "active"},
	})

	// Store a fresh token that won't trigger refresh (well beyond the 5-minute buffer).
	accessVaultID, _ := f.Vault.CreateSecret(t.Context(), f.TX, "access", []byte("fresh-token"))
	farFuture := time.Now().Add(2 * time.Hour)
	conn, _ := db.GetOAuthConnectionByProvider(t.Context(), f.TX, f.UserID, "google")
	_ = db.UpdateOAuthConnectionTokens(t.Context(), f.TX, conn.ID, f.UserID, accessVaultID, nil, &farFuture)

	result, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	tok, ok := capturedCreds.Get("access_token")
	if !ok {
		t.Fatal("expected access_token in credentials")
	}
	if tok != "fresh-token" {
		t.Errorf("expected access_token %q, got %q", "fresh-token", tok)
	}
}

func TestExecuteConnectorAction_OAuthPath_NilTokenExpirySkipsRefresh(t *testing.T) {
	t.Parallel()

	var capturedCreds connectors.Credentials
	f := setupOAuthExecutionTest(t, oauthExecOpts{
		OnExec:     func(creds connectors.Credentials) { capturedCreds = creds },
		Connection: &testhelper.OAuthConnectionOpts{Status: "active"},
	})

	// Store a token with no expiry information.
	accessVaultID, _ := f.Vault.CreateSecret(t.Context(), f.TX, "access", []byte("no-expiry-token"))
	conn, _ := db.GetOAuthConnectionByProvider(t.Context(), f.TX, f.UserID, "google")
	_ = db.UpdateOAuthConnectionTokens(t.Context(), f.TX, conn.ID, f.UserID, accessVaultID, nil, nil)

	result, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	tok, _ := capturedCreds.Get("access_token")
	if tok != "no-expiry-token" {
		t.Errorf("expected access_token %q, got %q", "no-expiry-token", tok)
	}
}

// TestExecuteConnectorAction_OAuthPath_NilExpiryNilOAuthRegistrySucceeds verifies
// that tokens with no expiry work even when deps.OAuthProviders is nil.
// This is the regression test for the OAuthProviders nil-check relocation.
func TestExecuteConnectorAction_OAuthPath_NilExpiryNilOAuthRegistrySucceeds(t *testing.T) {
	t.Parallel()

	var capturedCreds connectors.Credentials
	f := setupOAuthExecutionTest(t, oauthExecOpts{
		OnExec:          func(creds connectors.Credentials) { capturedCreds = creds },
		Connection:      &testhelper.OAuthConnectionOpts{Status: "active"},
		NoOAuthRegistry: true,
	})

	// Store a token with no expiry — refresh should never be attempted.
	accessVaultID, _ := f.Vault.CreateSecret(t.Context(), f.TX, "access", []byte("no-expiry-token"))
	conn, _ := db.GetOAuthConnectionByProvider(t.Context(), f.TX, f.UserID, "google")
	_ = db.UpdateOAuthConnectionTokens(t.Context(), f.TX, conn.ID, f.UserID, accessVaultID, nil, nil)

	result, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil, "")
	if err != nil {
		t.Fatalf("unexpected error with nil OAuthProviders and nil expiry: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	tok, _ := capturedCreds.Get("access_token")
	if tok != "no-expiry-token" {
		t.Errorf("expected access_token %q, got %q", "no-expiry-token", tok)
	}
}

// TestExecuteConnectorAction_OAuthPath_RefreshesExpiredToken is the hardest
// integration test: it wires up a real mock OAuth token server, stores an expired
// access token with a valid refresh token, and verifies the full refresh pipeline:
// token exchange → vault storage → DB update → fresh token passed to connector.
func TestExecuteConnectorAction_OAuthPath_RefreshesExpiredToken(t *testing.T) {
	t.Parallel()

	// Mock OAuth token endpoint that returns a new access token.
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "refreshed-access-token",
			"refresh_token": "rotated-refresh-token",
			"token_type":    "Bearer",
			"expires_in":    3600,
		})
	}))
	defer tokenSrv.Close()

	var capturedCreds connectors.Credentials
	f := setupOAuthExecutionTest(t, oauthExecOpts{
		OnExec:     func(creds connectors.Credentials) { capturedCreds = creds },
		Connection: &testhelper.OAuthConnectionOpts{Status: "active"},
	})

	// Override the OAuth registry to point at our mock token server.
	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID:           "google",
		TokenURL:     tokenSrv.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Source:       oauth.SourceBuiltIn,
	})
	f.Deps.OAuthProviders = oauthReg

	// Store an expired access token and a valid refresh token in the vault.
	accessVaultID, _ := f.Vault.CreateSecret(t.Context(), f.TX, "access", []byte("expired-access-token"))
	refreshVaultID, _ := f.Vault.CreateSecret(t.Context(), f.TX, "refresh", []byte("old-refresh-token"))

	pastExpiry := time.Now().Add(-10 * time.Minute)
	conn, _ := db.GetOAuthConnectionByProvider(t.Context(), f.TX, f.UserID, "google")
	_ = db.UpdateOAuthConnectionTokens(t.Context(), f.TX, conn.ID, f.UserID, accessVaultID, &refreshVaultID, &pastExpiry)

	result, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// The connector should receive the refreshed token, not the expired one.
	tok, ok := capturedCreds.Get("access_token")
	if !ok {
		t.Fatal("expected access_token in credentials")
	}
	if tok != "refreshed-access-token" {
		t.Errorf("expected refreshed token %q, got %q", "refreshed-access-token", tok)
	}

	// Verify the DB was updated with new token info.
	updated, err := db.GetOAuthConnectionByProvider(t.Context(), f.TX, f.UserID, "google")
	if err != nil {
		t.Fatalf("get connection after refresh: %v", err)
	}
	if updated.Status != db.OAuthStatusActive {
		t.Errorf("expected status to remain %q after successful refresh, got %q", db.OAuthStatusActive, updated.Status)
	}
	// The access token vault ID should have changed (new vault entry was created).
	if updated.AccessTokenVaultID == accessVaultID {
		t.Error("expected access_token_vault_id to change after refresh")
	}
}

// TestExecuteConnectorAction_OAuthPath_RefreshFailsTokenRevoked verifies that
// when the OAuth provider rejects the refresh token (e.g., revoked), the connection
// is marked needs_reauth and an OAuthRefreshError is returned.
func TestExecuteConnectorAction_OAuthPath_RefreshFailsTokenRevoked(t *testing.T) {
	t.Parallel()

	// Mock OAuth token endpoint that rejects the refresh.
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_grant",
			"error_description": "Token has been revoked",
		})
	}))
	defer tokenSrv.Close()

	f := setupOAuthExecutionTest(t, oauthExecOpts{
		Connection: &testhelper.OAuthConnectionOpts{Status: "active"},
	})

	// Override the OAuth registry to point at our mock token server.
	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID:           "google",
		TokenURL:     tokenSrv.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Source:       oauth.SourceBuiltIn,
	})
	f.Deps.OAuthProviders = oauthReg

	// Store an expired access token with a refresh token.
	accessVaultID, _ := f.Vault.CreateSecret(t.Context(), f.TX, "access", []byte("expired-token"))
	refreshVaultID, _ := f.Vault.CreateSecret(t.Context(), f.TX, "refresh", []byte("revoked-refresh-token"))
	pastExpiry := time.Now().Add(-10 * time.Minute)
	conn, _ := db.GetOAuthConnectionByProvider(t.Context(), f.TX, f.UserID, "google")
	_ = db.UpdateOAuthConnectionTokens(t.Context(), f.TX, conn.ID, f.UserID, accessVaultID, &refreshVaultID, &pastExpiry)

	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil, "")
	if err == nil {
		t.Fatal("expected error when refresh token is revoked")
	}
	if !connectors.IsOAuthRefreshError(err) {
		t.Errorf("expected OAuthRefreshError, got %T: %v", err, err)
	}

	// Verify connection was marked needs_reauth after failed refresh.
	updated, _ := db.GetOAuthConnectionByProvider(t.Context(), f.TX, f.UserID, "google")
	if updated.Status != db.OAuthStatusNeedsReauth {
		t.Errorf("expected status %q after failed refresh, got %q", db.OAuthStatusNeedsReauth, updated.Status)
	}
}

// ── resolveCredentialsWithFallback: no auto-resolve ─────────────────────────

func TestResolveCredentialsWithFallback_NoAgentID_ReturnsError(t *testing.T) {
	t.Parallel()

	// agentID=0 should always return an error — credentials require
	// an explicit agent binding.
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "trkno"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".do_thing", "Do Thing")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, connID, "api_key")

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore()}

	reqCreds, err := db.GetRequiredCredentialsByActionType(t.Context(), tx, connID+".do_thing")
	if err != nil {
		t.Fatalf("get creds: %v", err)
	}

	_, err = resolveCredentialsWithFallback(t.Context(), deps, 0, uid, connID+".do_thing", connID, "", reqCreds)
	if err == nil {
		t.Fatal("expected error when agentID is 0")
	}
	if !strings.Contains(err.Error(), "no credential assigned") {
		t.Errorf("expected 'no credential assigned' error, got: %v", err)
	}
}

func TestResolveCredentialsWithFallback_NoBinding_ReturnsError(t *testing.T) {
	t.Parallel()

	// Agent exists but has no credential binding — should return an error
	// instead of auto-resolving.
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "trknb"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".do_thing", "Do Thing")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, connID, "api_key")

	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore()}

	reqCreds, err := db.GetRequiredCredentialsByActionType(t.Context(), tx, connID+".do_thing")
	if err != nil {
		t.Fatalf("get creds: %v", err)
	}

	_, err = resolveCredentialsWithFallback(t.Context(), deps, agentID, uid, connID+".do_thing", connID, "", reqCreds)
	if err == nil {
		t.Fatal("expected error when no credential binding exists")
	}
	if !strings.Contains(err.Error(), "no credential assigned") {
		t.Errorf("expected 'no credential assigned' error, got: %v", err)
	}
}

func TestResolveCredentialsWithFallback_WithStaticBinding_Works(t *testing.T) {
	t.Parallel()

	// Agent has an explicit static credential binding — should resolve.
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "trksb"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".do_thing", "Do Thing")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, connID, "api_key")

	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	v := vault.NewMockVaultStore()
	credJSON, _ := json.Marshal(map[string]string{"api_key": "test-api-key"})
	vaultID, _ := v.CreateSecret(t.Context(), tx, "cred", credJSON)
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretID(t, tx, credID, uid, connID, vaultID)

	// Create an explicit binding.
	bindingID := testhelper.GenerateID(t, "accr_")
	_, err := db.UpsertAgentConnectorCredential(t.Context(), tx, db.UpsertAgentConnectorCredentialParams{
		ID: bindingID, AgentID: agentID, ConnectorID: connID,
		ApproverID: uid, CredentialID: &credID,
	})
	if err != nil {
		t.Fatalf("upsert binding: %v", err)
	}

	deps := &Deps{DB: tx, Vault: v}

	reqCreds, err := db.GetRequiredCredentialsByActionType(t.Context(), tx, connID+".do_thing")
	if err != nil {
		t.Fatalf("get creds: %v", err)
	}

	creds, err := resolveCredentialsWithFallback(t.Context(), deps, agentID, uid, connID+".do_thing", connID, "", reqCreds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	apiKey, ok := creds.Get("api_key")
	if !ok || apiKey != "test-api-key" {
		t.Errorf("expected api_key 'test-api-key', got %q (ok=%v)", apiKey, ok)
	}
}

func TestResolveCredentialsWithFallback_WithOAuthBinding_Works(t *testing.T) {
	t.Parallel()

	// Agent has an explicit OAuth connection binding — should resolve.
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "trkob"
	provider := connID
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".do_thing", "Do Thing")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, connID+"_oauth", provider, nil)

	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	v := vault.NewMockVaultStore()
	accessVaultID, _ := v.CreateSecret(t.Context(), tx, "access", []byte("oauth-access-token"))

	// Create an active OAuth connection.
	futureExpiry := time.Now().Add(1 * time.Hour)
	connOAuthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, provider, testhelper.OAuthConnectionOpts{
		Status: "active",
		Scopes: []string{},
	})
	conn, _ := db.GetOAuthConnectionByProvider(t.Context(), tx, uid, provider)
	_ = db.UpdateOAuthConnectionTokens(t.Context(), tx, conn.ID, uid, accessVaultID, nil, &futureExpiry)

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID: provider, AuthorizeURL: "https://example.com/auth", TokenURL: "https://example.com/token",
		ClientID: "cid", ClientSecret: "cs", Source: oauth.SourceBuiltIn,
	})

	// Create an explicit binding to the OAuth connection.
	bindingID := testhelper.GenerateID(t, "accr_")
	_, err := db.UpsertAgentConnectorCredential(t.Context(), tx, db.UpsertAgentConnectorCredentialParams{
		ID: bindingID, AgentID: agentID, ConnectorID: connID,
		ApproverID: uid, OAuthConnectionID: &conn.ID,
	})
	if err != nil {
		t.Fatalf("upsert binding: %v", err)
	}

	deps := &Deps{DB: tx, Vault: v, OAuthProviders: oauthReg}

	reqCreds, err := db.GetRequiredCredentialsByActionType(t.Context(), tx, connID+".do_thing")
	if err != nil {
		t.Fatalf("get creds: %v", err)
	}

	creds, err := resolveCredentialsWithFallback(t.Context(), deps, agentID, uid, connID+".do_thing", connID, "", reqCreds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tok, ok := creds.Get("access_token")
	if !ok || tok != "oauth-access-token" {
		t.Errorf("expected OAuth access_token, got %q (ok=%v)", tok, ok)
	}
}

// ── executeConnectorAction: static credential path with binding ─────────────

func TestExecuteConnectorAction_StaticPath_WithBinding(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testslack"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testslack.send_message", "Send Message")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "slack", "api_key")

	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	v := vault.NewMockVaultStore()
	credJSON, _ := json.Marshal(map[string]string{"api_key": "xoxb-test-token"})
	vaultID, err := v.CreateSecret(t.Context(), tx, "cred", credJSON)
	if err != nil {
		t.Fatalf("vault create: %v", err)
	}
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretID(t, tx, credID, uid, "slack", vaultID)

	// Create an explicit binding.
	bindingID := testhelper.GenerateID(t, "accr_")
	_, err = db.UpsertAgentConnectorCredential(t.Context(), tx, db.UpsertAgentConnectorCredentialParams{
		ID: bindingID, AgentID: agentID, ConnectorID: connID,
		ApproverID: uid, CredentialID: &credID,
	})
	if err != nil {
		t.Fatalf("upsert binding: %v", err)
	}

	var capturedCreds connectors.Credentials
	reg := connectors.NewRegistry()
	reg.Register(&credCapturingConnector{
		id:      connID,
		actions: []string{"testslack.send_message"},
		onExec:  func(creds connectors.Credentials) { capturedCreds = creds },
	})

	deps := &Deps{DB: tx, Vault: v, Connectors: reg}

	result, err := executeConnectorAction(t.Context(), deps, agentID, uid, "testslack.send_message", json.RawMessage(`{}`), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	tok, ok := capturedCreds.Get("api_key")
	if !ok {
		t.Fatal("expected api_key in credentials")
	}
	if tok != "xoxb-test-token" {
		t.Errorf("expected api_key %q, got %q", "xoxb-test-token", tok)
	}
}

func TestExecuteConnectorAction_UsesConnectorInstanceIDWhenProvided(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testslack2"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testslack2.send_message", "Send Message")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "slack", "api_key")

	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)
	inst, err := db.CreateAgentConnectorInstance(t.Context(), tx, db.CreateAgentConnectorInstanceParams{
		AgentID: agentID, ApproverID: uid, ConnectorID: connID, Label: "Other",
	})
	if err != nil {
		t.Fatalf("create instance: %v", err)
	}

	v := vault.NewMockVaultStore()
	credJSON, _ := json.Marshal(map[string]string{"api_key": "xoxb-instance-token"})
	vaultID, err := v.CreateSecret(t.Context(), tx, "cred", credJSON)
	if err != nil {
		t.Fatalf("vault create: %v", err)
	}
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretID(t, tx, credID, uid, "slack", vaultID)

	_, err = db.UpsertAgentConnectorCredentialByInstance(t.Context(), tx, db.UpsertAgentConnectorCredentialByInstanceParams{
		ID: testhelper.GenerateID(t, "accr_"), AgentID: agentID, ConnectorID: connID,
		ConnectorInstanceID: inst.ConnectorInstanceID, ApproverID: uid, CredentialID: &credID,
	})
	if err != nil {
		t.Fatalf("upsert binding: %v", err)
	}

	var capturedCreds connectors.Credentials
	reg := connectors.NewRegistry()
	reg.Register(&credCapturingConnector{
		id:      connID,
		actions: []string{"testslack2.send_message"},
		onExec:  func(creds connectors.Credentials) { capturedCreds = creds },
	})

	deps := &Deps{DB: tx, Vault: v, Connectors: reg}

	result, err := executeConnectorAction(t.Context(), deps, agentID, uid, "testslack2.send_message", json.RawMessage(`{}`), nil, inst.ConnectorInstanceID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	tok, ok := capturedCreds.Get("api_key")
	if !ok {
		t.Fatal("expected api_key in credentials")
	}
	if tok != "xoxb-instance-token" {
		t.Errorf("expected api_key %q, got %q", "xoxb-instance-token", tok)
	}
}

// ── executeConnectorAction: edge cases ──────────────────────────────────────

func TestExecuteConnectorAction_NilConnectorRegistry(t *testing.T) {
	t.Parallel()

	deps := &Deps{Connectors: nil}
	result, err := executeConnectorAction(context.Background(), deps, 0, "any-user", "any.action", json.RawMessage(`{}`), nil, "")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result != nil {
		t.Error("expected nil result when connector registry is nil")
	}
}

func TestExecuteConnectorAction_UnknownAction(t *testing.T) {
	t.Parallel()

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector("github", "github.create_issue"))

	deps := &Deps{Connectors: reg}
	result, err := executeConnectorAction(context.Background(), deps, 0, "any-user", "unknown.action", json.RawMessage(`{}`), nil, "")
	if err != nil {
		t.Fatalf("expected nil error for unknown action, got %v", err)
	}
	if result != nil {
		t.Error("expected nil result for unknown action")
	}
}

// ── handleConnectorError: OAuthRefreshError mapping ─────────────────────────

func TestHandleConnectorError_OAuthRefreshError_Returns401(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_test123"))

	oauthErr := &connectors.OAuthRefreshError{
		Provider: "google",
		Message:  "token refresh failed — user must re-authorize",
	}

	handled := handleConnectorError(w, r, oauthErr, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle OAuthRefreshError")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error.Code != ErrOAuthRefreshFailed {
		t.Errorf("expected error code %q, got %q", ErrOAuthRefreshFailed, resp.Error.Code)
	}
	if resp.Error.Details["provider"] != "google" {
		t.Errorf("expected provider detail %q, got %v", "google", resp.Error.Details["provider"])
	}
	// Verify the response includes action_required for agent UX.
	if resp.Error.Details["action_required"] == nil {
		t.Error("expected action_required in error details to guide the agent")
	}
}

func TestHandleConnectorError_OAuthRefreshError_Microsoft(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_msft"))

	oauthErr := &connectors.OAuthRefreshError{
		Provider: "microsoft",
		Message:  "refresh token expired",
	}

	handled := handleConnectorError(w, r, oauthErr, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle OAuthRefreshError")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}

	var resp ErrorResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if resp.Error.Details["provider"] != "microsoft" {
		t.Errorf("expected provider detail %q, got %v", "microsoft", resp.Error.Details["provider"])
	}
	if resp.Error.Retryable {
		t.Error("expected retryable=false for OAuth refresh errors")
	}
}

func TestHandleConnectorError_NonOAuthError_NotHandled(t *testing.T) {
	t.Parallel()

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_other"))

	handled := handleConnectorError(w, r, context.DeadlineExceeded, ConnectorContext{ActionType: "test.action"})
	if handled {
		t.Error("expected handleConnectorError to NOT handle generic errors")
	}
}

// ── handleConnectorError: ExternalError/AuthError/TimeoutError message surfacing ──

func TestHandleConnectorError_ExternalError_SurfacesMessage(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_ext"))

	extErr := &connectors.ExternalError{StatusCode: 404, Message: "Slack channel not found — verify the channel ID exists and the bot has access"}
	handled := handleConnectorError(w, r, extErr, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle ExternalError")
	}
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", w.Code)
	}
	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error.Message != extErr.Message {
		t.Errorf("expected message %q, got %q", extErr.Message, resp.Error.Message)
	}
}

func TestHandleConnectorError_AuthError_SurfacesMessage(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_auth"))

	authErr := &connectors.AuthError{Message: "GitHub API auth error (403): Resource not accessible by integration"}
	handled := handleConnectorError(w, r, authErr, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle AuthError")
	}
	if w.Code != http.StatusBadGateway {
		t.Errorf("expected status 502, got %d", w.Code)
	}
	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error.Message != authErr.Message {
		t.Errorf("expected message %q, got %q", authErr.Message, resp.Error.Message)
	}
}

func TestHandleConnectorError_TimeoutError_SurfacesMessage(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_timeout"))

	timeoutErr := &connectors.TimeoutError{Message: "Slack API request timed out: context deadline exceeded"}
	handled := handleConnectorError(w, r, timeoutErr, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle TimeoutError")
	}
	if w.Code != http.StatusGatewayTimeout {
		t.Errorf("expected status 504, got %d", w.Code)
	}
	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error.Message != timeoutErr.Message {
		t.Errorf("expected message %q, got %q", timeoutErr.Message, resp.Error.Message)
	}
}

// ── handleConnectorError: RateLimitError/ValidationError/OAuthRefreshError message surfacing ──

func TestHandleConnectorError_RateLimitError_SurfacesMessage(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_rl"))

	rlErr := &connectors.RateLimitError{Message: "GitHub API rate limit exceeded — resets in 42 minutes", RetryAfter: 42 * time.Minute}
	handled := handleConnectorError(w, r, rlErr, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle RateLimitError")
	}
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}
	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error.Message != rlErr.Message {
		t.Errorf("expected message %q, got %q", rlErr.Message, resp.Error.Message)
	}
}

func TestHandleConnectorError_ValidationError_SurfacesMessage(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_val"))

	valErr := &connectors.ValidationError{Message: "channel_id is required"}
	handled := handleConnectorError(w, r, valErr, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle ValidationError")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error.Message != valErr.Message {
		t.Errorf("expected message %q, got %q", valErr.Message, resp.Error.Message)
	}
}

func TestHandleConnectorError_OAuthRefreshError_SurfacesMessage(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_oauth"))

	oauthErr := &connectors.OAuthRefreshError{Provider: "google", Message: "Google OAuth token expired — user must re-authorize in Settings"}
	handled := handleConnectorError(w, r, oauthErr, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle OAuthRefreshError")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error.Message != oauthErr.Message {
		t.Errorf("expected message %q, got %q", oauthErr.Message, resp.Error.Message)
	}
}

// ── handleConnectorError: fallback messages when Message is empty ──

func TestHandleConnectorError_RateLimitError_FallbackMessage(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_rl_fallback"))

	rlErr := &connectors.RateLimitError{RetryAfter: 30 * time.Second} // no Message
	handled := handleConnectorError(w, r, rlErr, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle RateLimitError")
	}
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}
	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error.Message != "External service rate limited" {
		t.Errorf("expected fallback message, got %q", resp.Error.Message)
	}
}

func TestHandleConnectorError_ValidationError_FallbackMessage(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_val_fallback"))

	valErr := &connectors.ValidationError{} // no Message
	handled := handleConnectorError(w, r, valErr, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle ValidationError")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if resp.Error.Message != "Validation failed" {
		t.Errorf("expected fallback message, got %q", resp.Error.Message)
	}
}

func TestHandleConnectorError_OAuthRefreshError_FallbackMessage(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_oauth_fallback"))

	oauthErr := &connectors.OAuthRefreshError{Provider: "google"} // no Message
	handled := handleConnectorError(w, r, oauthErr, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle OAuthRefreshError")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected status 401, got %d", w.Code)
	}
	var resp ErrorResponse
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	expected := "OAuth authorization required — user must re-connect the provider in Settings"
	if resp.Error.Message != expected {
		t.Errorf("expected fallback message %q, got %q", expected, resp.Error.Message)
	}
}

// ── executeConnectorAction: payment method integration ──────────────────────

// paymentExecFixture holds common setup for payment method execution tests.
type paymentExecFixture struct {
	TX         db.DBTX
	UserID     string
	AgentID    int64
	ConnID     string
	ActionType string
	Vault      *vault.MockVaultStore
	Deps       *Deps
}

// paymentExecOpts configures setupPaymentExecTest.
type paymentExecOpts struct {
	// RequiresPayment controls whether the action requires a payment method.
	// Defaults to true.
	RequiresPayment *bool
	// Connector overrides the default stub connector registration.
	// When nil, a newTestStubConnector is used.
	Connector connectors.Connector
}

// setupPaymentExecTest creates a user, connector, action, credentials, and
// returns everything needed to call executeConnectorAction. Each call generates
// a unique connector ID to allow parallel tests without conflicts.
func setupPaymentExecTest(t *testing.T, opts paymentExecOpts) *paymentExecFixture {
	t.Helper()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "tpay_" + testhelper.GenerateID(t, "")[:8]
	actionType := connID + ".book"
	testhelper.InsertConnector(t, tx, connID)

	requiresPayment := true
	if opts.RequiresPayment != nil {
		requiresPayment = *opts.RequiresPayment
	}
	if requiresPayment {
		testhelper.InsertConnectorActionWithPayment(t, tx, connID, actionType, "Book")
	} else {
		testhelper.InsertConnectorAction(t, tx, connID, actionType, "Search")
	}
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, connID, "api_key")

	// Agent + agent_connector + credential binding.
	agentID := testhelper.InsertAgentWithStatus(t, tx, uid, "registered")
	testhelper.InsertAgentConnector(t, tx, agentID, uid, connID)

	v := vault.NewMockVaultStore()
	credJSON, _ := json.Marshal(map[string]string{"api_key": "test-key"})
	vaultID, _ := v.CreateSecret(t.Context(), tx, "cred", credJSON)
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretID(t, tx, credID, uid, connID, vaultID)

	// Create explicit credential binding.
	bindingID := testhelper.GenerateID(t, "accr_")
	_, bindErr := db.UpsertAgentConnectorCredential(t.Context(), tx, db.UpsertAgentConnectorCredentialParams{
		ID: bindingID, AgentID: agentID, ConnectorID: connID,
		ApproverID: uid, CredentialID: &credID,
	})
	if bindErr != nil {
		t.Fatalf("upsert binding: %v", bindErr)
	}

	reg := connectors.NewRegistry()
	if opts.Connector != nil {
		reg.Register(opts.Connector)
	} else {
		reg.Register(newTestStubConnector(connID, actionType))
	}

	return &paymentExecFixture{
		TX:         tx,
		UserID:     uid,
		AgentID:    agentID,
		ConnID:     connID,
		ActionType: actionType,
		Vault:      v,
		Deps:       &Deps{DB: tx, Vault: v, Connectors: reg},
	}
}

func TestExecuteConnectorAction_PaymentMethod_Success(t *testing.T) {
	t.Parallel()

	var capturedPayment *connectors.PaymentInfo
	// Use a temporary connID for the capturing connector — we need to know
	// it before calling setup, so we generate the connector inline.
	f := setupPaymentExecTest(t, paymentExecOpts{
		Connector: nil, // will be overridden below
	})

	// Replace the stub connector with a payment-capturing one.
	capConn := &paymentCapturingConnector{
		id:         f.ConnID,
		actionType: f.ActionType,
		onExec:     func(pi *connectors.PaymentInfo) { capturedPayment = pi },
	}
	reg := connectors.NewRegistry()
	reg.Register(capConn)
	f.Deps.Connectors = reg

	perTx := 10000
	monthly := 50000
	pmID := testhelper.InsertPaymentMethod(t, f.TX, f.UserID, testhelper.PaymentMethodOpts{
		PerTransactionLimit: &perTx,
		MonthlyLimit:        &monthly,
	})

	amount := 5000
	result, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, f.ActionType, json.RawMessage(`{}`), &paymentParams{
		PaymentMethodID: pmID,
		AmountCents:     &amount,
	}, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if capturedPayment == nil {
		t.Fatal("expected payment info to be passed to connector")
	}
	if capturedPayment.Brand != "visa" {
		t.Errorf("expected brand %q, got %q", "visa", capturedPayment.Brand)
	}
	if capturedPayment.Last4 != "4242" {
		t.Errorf("expected last4 %q, got %q", "4242", capturedPayment.Last4)
	}
	if capturedPayment.StripePaymentMethodID == "" {
		t.Error("expected non-empty StripePaymentMethodID")
	}
	if capturedPayment.AmountCents != 5000 {
		t.Errorf("expected AmountCents 5000, got %d", capturedPayment.AmountCents)
	}

	// Verify transaction was recorded (only on success).
	spend, err := db.GetMonthlySpend(t.Context(), f.TX, pmID)
	if err != nil {
		t.Fatalf("get monthly spend: %v", err)
	}
	if spend != 5000 {
		t.Errorf("expected monthly spend of 5000, got %d", spend)
	}
}

func TestExecuteConnectorAction_PaymentMethod_MissingRequired(t *testing.T) {
	t.Parallel()
	f := setupPaymentExecTest(t, paymentExecOpts{})

	// No payment params provided.
	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, f.ActionType, json.RawMessage(`{}`), nil, "")
	if err == nil {
		t.Fatal("expected error when payment_method_id is missing")
	}
	if !connectors.IsPaymentError(err) {
		t.Errorf("expected PaymentError, got %T: %v", err, err)
	}
	var pe *connectors.PaymentError
	if connectors.AsPaymentError(err, &pe) && pe.Code != connectors.PaymentErrMissing {
		t.Errorf("expected PaymentErrMissing, got %d", pe.Code)
	}
}

func TestExecuteConnectorAction_PaymentMethod_PerTxLimitExceeded(t *testing.T) {
	t.Parallel()
	f := setupPaymentExecTest(t, paymentExecOpts{})

	perTx := 1000
	pmID := testhelper.InsertPaymentMethod(t, f.TX, f.UserID, testhelper.PaymentMethodOpts{
		PerTransactionLimit: &perTx,
	})

	amount := 5000 // Exceeds 1000 limit
	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, f.ActionType, json.RawMessage(`{}`), &paymentParams{
		PaymentMethodID: pmID,
		AmountCents:     &amount,
	}, "")
	if err == nil {
		t.Fatal("expected error when per-transaction limit is exceeded")
	}
	var pe *connectors.PaymentError
	if !connectors.AsPaymentError(err, &pe) {
		t.Fatalf("expected PaymentError, got %T: %v", err, err)
	}
	if pe.Code != connectors.PaymentErrPerTxLimit {
		t.Errorf("expected PaymentErrPerTxLimit, got %d", pe.Code)
	}
	if pe.Details == nil {
		t.Fatal("expected Details to be populated for limit errors")
	}
	if pe.Details["requested_amount_cents"] != 5000 {
		t.Errorf("expected requested_amount_cents=5000, got %v", pe.Details["requested_amount_cents"])
	}
	if pe.Details["per_transaction_limit_cents"] != 1000 {
		t.Errorf("expected per_transaction_limit_cents=1000, got %v", pe.Details["per_transaction_limit_cents"])
	}
}

func TestExecuteConnectorAction_PaymentMethod_MonthlyLimitExceeded(t *testing.T) {
	t.Parallel()
	f := setupPaymentExecTest(t, paymentExecOpts{})

	monthly := 10000
	pmID := testhelper.InsertPaymentMethod(t, f.TX, f.UserID, testhelper.PaymentMethodOpts{
		MonthlyLimit: &monthly,
	})

	// Pre-seed a transaction to use up most of the monthly limit.
	_, err := db.CreatePaymentMethodTransaction(t.Context(), f.TX, &db.PaymentMethodTransaction{
		PaymentMethodID: pmID,
		UserID:          f.UserID,
		ConnectorID:     f.ConnID,
		ActionType:      f.ActionType,
		AmountCents:     9000,
		Description:     "prior tx",
	})
	if err != nil {
		t.Fatalf("seed transaction: %v", err)
	}

	amount := 2000 // 9000 + 2000 = 11000 > 10000 monthly limit
	_, err = executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, f.ActionType, json.RawMessage(`{}`), &paymentParams{
		PaymentMethodID: pmID,
		AmountCents:     &amount,
	}, "")
	if err == nil {
		t.Fatal("expected error when monthly limit is exceeded")
	}
	var pe *connectors.PaymentError
	if !connectors.AsPaymentError(err, &pe) {
		t.Fatalf("expected PaymentError, got %T: %v", err, err)
	}
	if pe.Code != connectors.PaymentErrMonthlyLimit {
		t.Errorf("expected PaymentErrMonthlyLimit, got %d", pe.Code)
	}
	if pe.Details == nil {
		t.Fatal("expected Details to be populated for monthly limit errors")
	}
	if pe.Details["requested_amount_cents"] != 2000 {
		t.Errorf("expected requested_amount_cents=2000, got %v", pe.Details["requested_amount_cents"])
	}
	if pe.Details["monthly_limit_cents"] != 10000 {
		t.Errorf("expected monthly_limit_cents=10000, got %v", pe.Details["monthly_limit_cents"])
	}
	if pe.Details["current_spend_cents"] != 9000 {
		t.Errorf("expected current_spend_cents=9000, got %v", pe.Details["current_spend_cents"])
	}
	if pe.Details["remaining_cents"] != 1000 {
		t.Errorf("expected remaining_cents=1000, got %v", pe.Details["remaining_cents"])
	}
}

func TestExecuteConnectorAction_PaymentMethod_NotFound(t *testing.T) {
	t.Parallel()
	f := setupPaymentExecTest(t, paymentExecOpts{})

	amount := 100
	_, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, f.ActionType, json.RawMessage(`{}`), &paymentParams{
		PaymentMethodID: "00000000-0000-0000-0000-000000000099",
		AmountCents:     &amount,
	}, "")
	if err == nil {
		t.Fatal("expected error for nonexistent payment method")
	}
	var pe *connectors.PaymentError
	if !connectors.AsPaymentError(err, &pe) {
		t.Fatalf("expected PaymentError, got %T: %v", err, err)
	}
	if pe.Code != connectors.PaymentErrNotFound {
		t.Errorf("expected PaymentErrNotFound, got %d", pe.Code)
	}
}

func TestExecuteConnectorAction_NoPaymentMethod_WhenNotRequired(t *testing.T) {
	t.Parallel()
	noPayment := false
	f := setupPaymentExecTest(t, paymentExecOpts{RequiresPayment: &noPayment})

	// No payment params — should succeed because action doesn't require payment.
	result, err := executeConnectorAction(t.Context(), f.Deps, f.AgentID, f.UserID, f.ActionType, json.RawMessage(`{}`), nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
}

// paymentCapturingConnector is a test connector that captures PaymentInfo.
type paymentCapturingConnector struct {
	id         string
	actionType string
	onExec     func(*connectors.PaymentInfo)
}

func (c *paymentCapturingConnector) ID() string { return c.id }
func (c *paymentCapturingConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		c.actionType: &paymentCapturingAction{onExec: c.onExec},
	}
}
func (c *paymentCapturingConnector) ValidateCredentials(_ context.Context, _ connectors.Credentials) error {
	return nil
}

type paymentCapturingAction struct {
	onExec func(*connectors.PaymentInfo)
}

func (a *paymentCapturingAction) Execute(_ context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	if a.onExec != nil {
		a.onExec(req.Payment)
	}
	return &connectors.ActionResult{}, nil
}

// ── handleConnectorError: PaymentError mapping ──────────────────────────────

func TestHandleConnectorError_PaymentError_Missing(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_pm1"))

	pe := &connectors.PaymentError{Code: connectors.PaymentErrMissing, Message: "payment_method_id required"}
	handled := handleConnectorError(w, r, pe, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle PaymentError")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestHandleConnectorError_PaymentError_LimitExceeded(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_pm2"))

	pe := &connectors.PaymentError{Code: connectors.PaymentErrPerTxLimit, Message: "exceeds limit"}
	handled := handleConnectorError(w, r, pe, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle PaymentError")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("expected status 403, got %d", w.Code)
	}
}

func TestHandleConnectorError_PaymentError_InvalidAmount(t *testing.T) {
	t.Parallel()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/approvals/request", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_pm3"))

	pe := &connectors.PaymentError{Code: connectors.PaymentErrInvalidAmount, Message: "amount_cents must be non-negative"}
	handled := handleConnectorError(w, r, pe, ConnectorContext{ActionType: "test.action"})
	if !handled {
		t.Fatal("expected handleConnectorError to handle PaymentError")
	}
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
	// Verify it uses ErrInvalidRequest, not ErrPaymentMethodRequired
	var resp map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	errObj, ok := resp["error"].(map[string]any)
	if !ok {
		t.Fatal("expected error object in response")
	}
	if code, _ := errObj["code"].(string); code != string(ErrInvalidRequest) {
		t.Errorf("expected error code %q, got %q", ErrInvalidRequest, code)
	}
}

func TestCredentialsFromOAuthConnection_SlackStripsLegacyUserVaultKey(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	v := vault.NewMockVaultStore()
	vaultID := "aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"
	v.SeedSecretForTest(vaultID, []byte("xoxp-secret"))

	connID := testhelper.GenerateID(t, "oconn_")
	extra := []byte(`{"user_access_token_vault_id":"should-not-leak","team_name":"Acme"}`)
	testhelper.InsertOAuthConnectionFull(t, tx, connID, uid, "slack", testhelper.OAuthConnectionOpts{
		AccessTokenVaultID: vaultID,
		Scopes:             []string{"chat:write"},
		ExtraData:          extra,
	})

	conn, err := db.GetOAuthConnectionByProvider(t.Context(), tx, uid, "slack")
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	if conn == nil {
		t.Fatal("expected oauth connection")
	}

	deps := &Deps{DB: tx, Vault: v}
	creds, err := credentialsFromOAuthConnection(t.Context(), deps, conn)
	if err != nil {
		t.Fatalf("credentialsFromOAuthConnection: %v", err)
	}

	if _, ok := creds.Get("user_access_token_vault_id"); ok {
		t.Error("legacy user_access_token_vault_id must not be passed to connectors for Slack")
	}
	team, _ := creds.Get("team_name")
	if team != "Acme" {
		t.Errorf("team_name = %q, want Acme", team)
	}
	tok, _ := creds.Get("access_token")
	if tok != "xoxp-secret" {
		t.Errorf("access_token = %q, want xoxp-secret", tok)
	}
}
