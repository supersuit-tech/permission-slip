package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
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

	result, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`))
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

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error when no OAuth connection exists")
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

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`))
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

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.list_emails", json.RawMessage(`{}`))
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

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error when vault is nil")
	}
}

func TestExecuteConnectorAction_OAuthPath_NoOAuthRegistry(t *testing.T) {
	t.Parallel()

	f := setupOAuthExecutionTest(t, oauthExecOpts{NoOAuthRegistry: true})

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`))
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

	_, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`))
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

	result, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`))
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

	result, err := executeConnectorAction(t.Context(), f.Deps, f.UserID, "testgoogle.send_email", json.RawMessage(`{}`))
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

	result, err := executeConnectorAction(t.Context(), deps, uid, "testslack.send_message", json.RawMessage(`{}`))
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

// ── executeConnectorAction: edge cases ──────────────────────────────────────

func TestExecuteConnectorAction_NilConnectorRegistry(t *testing.T) {
	t.Parallel()

	deps := &Deps{Connectors: nil}
	result, err := executeConnectorAction(context.Background(), deps, "any-user", "any.action", json.RawMessage(`{}`))
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
	result, err := executeConnectorAction(context.Background(), deps, "any-user", "unknown.action", json.RawMessage(`{}`))
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
