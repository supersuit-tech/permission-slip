package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
	"github.com/supersuit-tech/permission-slip-web/oauth"
	"github.com/supersuit-tech/permission-slip-web/vault"
)

// ── test helpers ────────────────────────────────────────────────────────────

// oauthExecFixture holds the common setup for OAuth execution tests.
type oauthExecFixture struct {
	TX       db.DBTX
	Deps     *Deps
	UserID   string
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

	// Vault.
	v := vault.NewMockVaultStore()

	// OAuth connection (optional).
	if opts.Connection != nil {
		connOAuthID := testhelper.GenerateID(t, "oconn_")
		// Default scopes to empty array to satisfy NOT NULL constraint.
		if opts.Connection.Scopes == nil {
			opts.Connection.Scopes = []string{}
		}
		testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, provider, *opts.Connection)
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
			Scopes:  []string{"https://www.googleapis.com/auth/gmail.send"},
			Status:  "active",
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

	result, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil)
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

func TestExecuteConnectorAction_OAuthPath_NoConnection(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{
		// No Connection — tests the "user hasn't connected" case.
	})

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil)
	if err == nil {
		t.Fatal("expected error when no OAuth connection exists")
	}
	if !connectors.IsOAuthRefreshError(err) {
		t.Errorf("expected OAuthRefreshError, got %T: %v", err, err)
	}
	// Verify the error message mentions the provider so agents can display helpful info.
	if !strings.Contains(err.Error(), "google") {
		t.Errorf("expected error to mention provider name, got: %s", err.Error())
	}
}

func TestExecuteConnectorAction_OAuthPath_NeedsReauth(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{
		Connection: &testhelper.OAuthConnectionOpts{Status: "needs_reauth"},
	})

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil)
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

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.list_emails", json.RawMessage(`{}`), nil)
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

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil)
	if err == nil {
		t.Fatal("expected error when vault is nil")
	}
}

func TestExecuteConnectorAction_OAuthPath_NoOAuthRegistry(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{NoOAuthRegistry: true})

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil)
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

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil)
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

	result, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil)
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

	result, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil)
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

	result, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil)
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

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`), nil)
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

// ── resolveCredentialsWithFallback: multi-credential fallback ────────────────

func TestResolveCredentialsWithFallback_OAuthFallsBackToAPIKey(t *testing.T) {
	t.Parallel()

	// Set up a connector with BOTH OAuth and API key credentials, but
	// no OAuth connection — should fall back to the API key.
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "trkfb"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".do_thing", "Do Thing")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, connID+"_oauth", connID, nil)
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, connID, "api_key")

	v := vault.NewMockVaultStore()
	credJSON, _ := json.Marshal(map[string]string{"api_key": "test-api-key"})
	vaultID, _ := v.CreateSecret(t.Context(), tx, "cred", credJSON)
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretID(t, tx, credID, uid, connID, vaultID)

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID: connID, AuthorizeURL: "https://example.com/auth", TokenURL: "https://example.com/token",
		ClientID: "cid", ClientSecret: "cs", Source: oauth.SourceBuiltIn,
	})

	deps := &Deps{DB: tx, Vault: v, OAuthProviders: oauthReg}

	reqCreds, err := db.GetRequiredCredentialsByActionType(t.Context(), tx, connID+".do_thing")
	if err != nil {
		t.Fatalf("get creds: %v", err)
	}

	creds, err := resolveCredentialsWithFallback(t.Context(), deps, uid, connID+".do_thing", connID, reqCreds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	apiKey, ok := creds.Get("api_key")
	if !ok || apiKey != "test-api-key" {
		t.Errorf("expected api_key 'test-api-key', got %q (ok=%v)", apiKey, ok)
	}
}

func TestResolveCredentialsWithFallback_OAuthSucceedsWhenAvailable(t *testing.T) {
	t.Parallel()

	// Set up a connector with BOTH OAuth and API key, WITH an active
	// OAuth connection — should use OAuth, not fall back.
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "trkoau"
	provider := connID
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".do_thing", "Do Thing")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, connID+"_oauth", provider, nil)
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, connID, "api_key")

	v := vault.NewMockVaultStore()
	accessVaultID, _ := v.CreateSecret(t.Context(), tx, "access", []byte("oauth-access-token"))

	// Create an active OAuth connection.
	futureExpiry := time.Now().Add(1 * time.Hour)
	connOAuthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, provider, testhelper.OAuthConnectionOpts{
		Status:  "active",
		Scopes:  []string{},
	})
	conn, _ := db.GetOAuthConnectionByProvider(t.Context(), tx, uid, provider)
	_ = db.UpdateOAuthConnectionTokens(t.Context(), tx, conn.ID, uid, accessVaultID, nil, &futureExpiry)

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID: provider, AuthorizeURL: "https://example.com/auth", TokenURL: "https://example.com/token",
		ClientID: "cid", ClientSecret: "cs", Source: oauth.SourceBuiltIn,
	})

	deps := &Deps{DB: tx, Vault: v, OAuthProviders: oauthReg}

	reqCreds, err := db.GetRequiredCredentialsByActionType(t.Context(), tx, connID+".do_thing")
	if err != nil {
		t.Fatalf("get creds: %v", err)
	}

	creds, err := resolveCredentialsWithFallback(t.Context(), deps, uid, connID+".do_thing", connID, reqCreds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tok, ok := creds.Get("access_token")
	if !ok || tok != "oauth-access-token" {
		t.Errorf("expected OAuth access_token, got %q (ok=%v)", tok, ok)
	}
}

func TestResolveCredentialsWithFallback_BothFail_ReturnsError(t *testing.T) {
	t.Parallel()

	// No OAuth connection AND no API key stored — both fail.
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "trkbf"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".do_thing", "Do Thing")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, connID+"_oauth", connID, nil)
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, connID, "api_key")

	v := vault.NewMockVaultStore()
	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID: connID, AuthorizeURL: "https://example.com/auth", TokenURL: "https://example.com/token",
		ClientID: "cid", ClientSecret: "cs", Source: oauth.SourceBuiltIn,
	})

	deps := &Deps{DB: tx, Vault: v, OAuthProviders: oauthReg}

	reqCreds, err := db.GetRequiredCredentialsByActionType(t.Context(), tx, connID+".do_thing")
	if err != nil {
		t.Fatalf("get creds: %v", err)
	}

	_, err = resolveCredentialsWithFallback(t.Context(), deps, uid, connID+".do_thing", connID, reqCreds)
	if err == nil {
		t.Fatal("expected error when both auth methods fail")
	}
}

// ── executeConnectorAction: static credential path ──────────────────────────

func TestExecuteConnectorAction_StaticPath_StillWorks(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testslack"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testslack.send_message", "Send Message")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "slack", "api_key")

	v := vault.NewMockVaultStore()
	credJSON, _ := json.Marshal(map[string]string{"api_key": "xoxb-test-token"})
	vaultID, err := v.CreateSecret(t.Context(), tx, "cred", credJSON)
	if err != nil {
		t.Fatalf("vault create: %v", err)
	}
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretID(t, tx, credID, uid, "slack", vaultID)

	var capturedCreds connectors.Credentials
	reg := connectors.NewRegistry()
	reg.Register(&credCapturingConnector{
		id:      connID,
		actions: []string{"testslack.send_message"},
		onExec:  func(creds connectors.Credentials) { capturedCreds = creds },
	})

	deps := &Deps{DB: tx, Vault: v, Connectors: reg}

	result, err := executeConnectorAction(t.Context(), deps, uid, "testslack.send_message", json.RawMessage(`{}`), nil)
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

// ── Implicit OAuth provider fallback tests ──────────────────────────────────
// These test the code path where a connector declares only static credentials
// (api_key) but has a matching built-in OAuth provider in the registry.
// resolveCredentialsWithFallback should synthesize an oauth2 entry and try
// OAuth first, falling back to static credentials if OAuth isn't available.

func TestResolveCredentialsWithFallback_ImplicitOAuth_UsesOAuthWhenConnected(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testshopify"
	provider := "testshopify"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".list_products", "List Products")
	// Only declare api_key — no explicit oauth2 credential.
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "shopify_api_key", "api_key")

	v := vault.NewMockVaultStore()
	accessToken, err := v.CreateSecret(t.Context(), tx, "access", []byte("shpat_oauth_token"))
	if err != nil {
		t.Fatalf("vault create: %v", err)
	}

	// Create an active OAuth connection for the provider.
	connOAuthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, provider, testhelper.OAuthConnectionOpts{
		Status:              "active",
		Scopes:              []string{"write_products"},
		AccessTokenVaultID:  accessToken,
		TokenExpiry:         func() *time.Time { t := time.Now().Add(time.Hour); return &t }(),
	})

	// Register a matching OAuth provider.
	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID:           provider,
		AuthorizeURL: "https://test.myshopify.com/admin/oauth/authorize",
		TokenURL:     "https://test.myshopify.com/admin/oauth/access_token",
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Source:       oauth.SourceBuiltIn,
	})

	reqCreds := []db.RequiredCredential{{Service: "shopify_api_key", AuthType: "api_key"}}
	deps := &Deps{DB: tx, Vault: v, OAuthProviders: oauthReg}

	creds, err := resolveCredentialsWithFallback(t.Context(), deps, uid, connID+".list_products", connID, reqCreds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tok, ok := creds.Get("access_token")
	if !ok {
		t.Fatal("expected access_token in credentials (from OAuth)")
	}
	if tok != "shpat_oauth_token" {
		t.Errorf("expected OAuth token %q, got %q", "shpat_oauth_token", tok)
	}
}

func TestResolveCredentialsWithFallback_ImplicitOAuth_FallsBackToStatic(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testshopify"
	provider := "testshopify"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".list_products", "List Products")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "shopify_api_key", "api_key")

	v := vault.NewMockVaultStore()
	// Store a static API key credential (no OAuth connection).
	credJSON, _ := json.Marshal(map[string]string{"api_key": "shpat_static_key"})
	vaultID, err := v.CreateSecret(t.Context(), tx, "cred", credJSON)
	if err != nil {
		t.Fatalf("vault create: %v", err)
	}
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretID(t, tx, credID, uid, "shopify_api_key", vaultID)

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID:           provider,
		AuthorizeURL: "https://test.myshopify.com/admin/oauth/authorize",
		TokenURL:     "https://test.myshopify.com/admin/oauth/access_token",
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Source:       oauth.SourceBuiltIn,
	})

	reqCreds := []db.RequiredCredential{{Service: "shopify_api_key", AuthType: "api_key"}}
	deps := &Deps{DB: tx, Vault: v, OAuthProviders: oauthReg}

	creds, err := resolveCredentialsWithFallback(t.Context(), deps, uid, connID+".list_products", connID, reqCreds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tok, ok := creds.Get("api_key")
	if !ok {
		t.Fatal("expected api_key in credentials (static fallback)")
	}
	if tok != "shpat_static_key" {
		t.Errorf("expected static key %q, got %q", "shpat_static_key", tok)
	}
}

func TestResolveCredentialsWithFallback_ImplicitOAuth_NeedsReauthFallsBack(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testshopify"
	provider := "testshopify"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, connID+".list_products", "List Products")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "shopify_api_key", "api_key")

	v := vault.NewMockVaultStore()

	// Create a needs_reauth OAuth connection.
	connOAuthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, provider, testhelper.OAuthConnectionOpts{
		Status: "needs_reauth",
		Scopes: []string{"write_products"},
	})

	// Store a static API key credential as fallback.
	credJSON, _ := json.Marshal(map[string]string{"api_key": "shpat_static_key"})
	vaultID, err := v.CreateSecret(t.Context(), tx, "cred", credJSON)
	if err != nil {
		t.Fatalf("vault create: %v", err)
	}
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretID(t, tx, credID, uid, "shopify_api_key", vaultID)

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID:           provider,
		AuthorizeURL: "https://test.myshopify.com/admin/oauth/authorize",
		TokenURL:     "https://test.myshopify.com/admin/oauth/access_token",
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Source:       oauth.SourceBuiltIn,
	})

	reqCreds := []db.RequiredCredential{{Service: "shopify_api_key", AuthType: "api_key"}}
	deps := &Deps{DB: tx, Vault: v, OAuthProviders: oauthReg}

	creds, err := resolveCredentialsWithFallback(t.Context(), deps, uid, connID+".list_products", connID, reqCreds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	tok, ok := creds.Get("api_key")
	if !ok {
		t.Fatal("expected api_key in credentials (static fallback after needs_reauth)")
	}
	if tok != "shpat_static_key" {
		t.Errorf("expected static key %q, got %q", "shpat_static_key", tok)
	}
}

// ── executeConnectorAction: edge cases ──────────────────────────────────────

func TestExecuteConnectorAction_NilConnectorRegistry(t *testing.T) {
	t.Parallel()

	deps := &Deps{Connectors: nil}
	result, err := executeConnectorAction(context.Background(), deps, "any-user", "any.action", json.RawMessage(`{}`), nil)
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
	result, err := executeConnectorAction(context.Background(), deps, "any-user", "unknown.action", json.RawMessage(`{}`), nil)
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
	r := httptest.NewRequest(http.MethodPost, "/actions/execute", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_test123"))

	oauthErr := &connectors.OAuthRefreshError{
		Provider: "google",
		Message:  "token refresh failed — user must re-authorize",
	}

	handled := handleConnectorError(w, r, oauthErr)
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
	r := httptest.NewRequest(http.MethodPost, "/actions/execute", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_msft"))

	oauthErr := &connectors.OAuthRefreshError{
		Provider: "microsoft",
		Message:  "refresh token expired",
	}

	handled := handleConnectorError(w, r, oauthErr)
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
	r := httptest.NewRequest(http.MethodPost, "/actions/execute", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_other"))

	handled := handleConnectorError(w, r, context.DeadlineExceeded)
	if handled {
		t.Error("expected handleConnectorError to NOT handle generic errors")
	}
}

// ── executeConnectorAction: payment method integration ──────────────────────

// paymentExecFixture holds common setup for payment method execution tests.
type paymentExecFixture struct {
	TX         db.DBTX
	UserID     string
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

	v := vault.NewMockVaultStore()
	credJSON, _ := json.Marshal(map[string]string{"api_key": "test-key"})
	vaultID, _ := v.CreateSecret(t.Context(), tx, "cred", credJSON)
	credID := testhelper.GenerateID(t, "cred_")
	testhelper.InsertCredentialWithVaultSecretID(t, tx, credID, uid, connID, vaultID)

	reg := connectors.NewRegistry()
	if opts.Connector != nil {
		reg.Register(opts.Connector)
	} else {
		reg.Register(newTestStubConnector(connID, actionType))
	}

	return &paymentExecFixture{
		TX:         tx,
		UserID:     uid,
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
	result, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, f.ActionType, json.RawMessage(`{}`), &paymentParams{
		PaymentMethodID: pmID,
		AmountCents:     &amount,
	})
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
	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, f.ActionType, json.RawMessage(`{}`), nil)
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
	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, f.ActionType, json.RawMessage(`{}`), &paymentParams{
		PaymentMethodID: pmID,
		AmountCents:     &amount,
	})
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
	_, err = executeConnectorAction(t.Context(), f.Deps, f.UserID, f.ActionType, json.RawMessage(`{}`), &paymentParams{
		PaymentMethodID: pmID,
		AmountCents:     &amount,
	})
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
	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, f.ActionType, json.RawMessage(`{}`), &paymentParams{
		PaymentMethodID: "00000000-0000-0000-0000-000000000099",
		AmountCents:     &amount,
	})
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
	result, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, f.ActionType, json.RawMessage(`{}`), nil)
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
	r := httptest.NewRequest(http.MethodPost, "/actions/execute", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_pm1"))

	pe := &connectors.PaymentError{Code: connectors.PaymentErrMissing, Message: "payment_method_id required"}
	handled := handleConnectorError(w, r, pe)
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
	r := httptest.NewRequest(http.MethodPost, "/actions/execute", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_pm2"))

	pe := &connectors.PaymentError{Code: connectors.PaymentErrPerTxLimit, Message: "exceeds limit"}
	handled := handleConnectorError(w, r, pe)
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
	r := httptest.NewRequest(http.MethodPost, "/actions/execute", nil)
	r = r.WithContext(context.WithValue(r.Context(), traceIDKey{}, "trace_pm3"))

	pe := &connectors.PaymentError{Code: connectors.PaymentErrInvalidAmount, Message: "amount_cents must be non-negative"}
	handled := handleConnectorError(w, r, pe)
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
