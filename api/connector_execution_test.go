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

// ── executeConnectorAction: OAuth path ──────────────────────────────────────

func TestExecuteConnectorAction_OAuthPath_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Set up connector with OAuth-type credential in DB.
	connID := "testgoogle"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testgoogle.send_email", "Send Email")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", []string{"https://www.googleapis.com/auth/gmail.send"})

	// Store access token in vault and create OAuth connection.
	v := vault.NewMockVaultStore()
	accessVaultID, err := v.CreateSecret(t.Context(), tx, "access", []byte("valid-access-token"))
	if err != nil {
		t.Fatalf("vault create: %v", err)
	}
	futureExpiry := time.Now().Add(1 * time.Hour)
	connOAuthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, "google", testhelper.OAuthConnectionOpts{
		AccessTokenVaultID: accessVaultID,
		Scopes:             []string{"https://www.googleapis.com/auth/gmail.send"},
		TokenExpiry:        &futureExpiry,
		Status:             "active",
	})

	// Set up connector registry with a stub that captures the credentials.
	var capturedCreds connectors.Credentials
	reg := connectors.NewRegistry()
	reg.Register(&credCapturingConnector{
		id:      connID,
		actions: []string{"testgoogle.send_email"},
		onExec:  func(creds connectors.Credentials) { capturedCreds = creds },
	})

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID:           "google",
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		Scopes:       []string{"openid"},
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

	result, err := executeConnectorAction(t.Context(), deps, uid, "testgoogle.send_email", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Verify the connector received the access token.
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
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testgoogle"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testgoogle.send_email", "Send Email")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", []string{"https://www.googleapis.com/auth/gmail.send"})

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, "testgoogle.send_email"))

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID:           "google",
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		ClientID:     "test-id",
		ClientSecret: "test-secret",
		Source:       oauth.SourceBuiltIn,
	})

	deps := &Deps{
		DB:             tx,
		Vault:          vault.NewMockVaultStore(),
		Connectors:     reg,
		OAuthProviders: oauthReg,
	}

	// No OAuth connection exists for this user/provider.
	_, err := executeConnectorAction(t.Context(), deps, uid, "testgoogle.send_email", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error when no OAuth connection exists")
	}
	if !connectors.IsOAuthRefreshError(err) {
		t.Errorf("expected OAuthRefreshError, got %T: %v", err, err)
	}
}

func TestExecuteConnectorAction_OAuthPath_NeedsReauth(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testgoogle"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testgoogle.send_email", "Send Email")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", nil)

	// Create connection with needs_reauth status.
	connOAuthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, "google", testhelper.OAuthConnectionOpts{
		Scopes: []string{},
		Status: "needs_reauth",
	})

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, "testgoogle.send_email"))

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID: "google", AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL: "https://oauth2.googleapis.com/token",
		ClientID: "id", ClientSecret: "secret", Source: oauth.SourceBuiltIn,
	})

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), Connectors: reg, OAuthProviders: oauthReg}

	_, err := executeConnectorAction(t.Context(), deps, uid, "testgoogle.send_email", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for needs_reauth connection")
	}
	if !connectors.IsOAuthRefreshError(err) {
		t.Errorf("expected OAuthRefreshError, got %T: %v", err, err)
	}
}

func TestExecuteConnectorAction_OAuthPath_RevokedConnection(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testgoogle"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testgoogle.list_emails", "List Emails")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", nil)

	connOAuthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, "google", testhelper.OAuthConnectionOpts{
		Scopes: []string{},
		Status: "revoked",
	})

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, "testgoogle.list_emails"))

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID: "google", AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL: "https://oauth2.googleapis.com/token",
		ClientID: "id", ClientSecret: "secret", Source: oauth.SourceBuiltIn,
	})

	deps := &Deps{DB: tx, Vault: vault.NewMockVaultStore(), Connectors: reg, OAuthProviders: oauthReg}

	_, err := executeConnectorAction(t.Context(), deps, uid, "testgoogle.list_emails", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for revoked connection")
	}
	if !connectors.IsOAuthRefreshError(err) {
		t.Errorf("expected OAuthRefreshError, got %T: %v", err, err)
	}
}

func TestExecuteConnectorAction_OAuthPath_NoVault(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testgoogle"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testgoogle.send_email", "Send Email")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", nil)

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, "testgoogle.send_email"))

	deps := &Deps{
		DB:             tx,
		Vault:          nil, // no vault
		Connectors:     reg,
		OAuthProviders: oauth.NewRegistry(),
	}

	_, err := executeConnectorAction(t.Context(), deps, uid, "testgoogle.send_email", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error when vault is nil")
	}
}

func TestExecuteConnectorAction_OAuthPath_NoOAuthRegistry(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testgoogle"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testgoogle.send_email", "Send Email")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", nil)

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, "testgoogle.send_email"))

	deps := &Deps{
		DB:             tx,
		Vault:          vault.NewMockVaultStore(),
		Connectors:     reg,
		OAuthProviders: nil, // no registry
	}

	_, err := executeConnectorAction(t.Context(), deps, uid, "testgoogle.send_email", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error when OAuth registry is nil")
	}
}

func TestExecuteConnectorAction_StaticPath_StillWorks(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Set up connector with api_key auth type.
	connID := "testslack"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testslack.send_message", "Send Message")
	testhelper.InsertConnectorRequiredCredential(t, tx, connID, "slack", "api_key")

	// Store static credential in vault and link it to the credentials table.
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

	deps := &Deps{
		DB:         tx,
		Vault:      v,
		Connectors: reg,
	}

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

func TestExecuteConnectorAction_OAuthPath_ExpiredTokenNoRefreshToken(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testgoogle"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testgoogle.send_email", "Send Email")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", nil)

	v := vault.NewMockVaultStore()
	accessVaultID, _ := v.CreateSecret(t.Context(), tx, "access", []byte("expired-token"))
	pastExpiry := time.Now().Add(-10 * time.Minute)
	connOAuthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, "google", testhelper.OAuthConnectionOpts{
		AccessTokenVaultID: accessVaultID,
		Scopes:             []string{},
		TokenExpiry:        &pastExpiry,
		Status:             "active",
		// No refresh token
	})

	reg := connectors.NewRegistry()
	reg.Register(newTestStubConnector(connID, "testgoogle.send_email"))

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID: "google", AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL: "https://oauth2.googleapis.com/token",
		ClientID: "id", ClientSecret: "secret", Source: oauth.SourceBuiltIn,
	})

	deps := &Deps{DB: tx, Vault: v, Connectors: reg, OAuthProviders: oauthReg}

	_, err := executeConnectorAction(t.Context(), deps, uid, "testgoogle.send_email", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("expected error for expired token without refresh token")
	}
	if !connectors.IsOAuthRefreshError(err) {
		t.Errorf("expected OAuthRefreshError, got %T: %v", err, err)
	}

	// Verify connection status was updated to needs_reauth.
	conn, err := db.GetOAuthConnectionByProvider(t.Context(), tx, uid, "google")
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	if conn.Status != db.OAuthStatusNeedsReauth {
		t.Errorf("expected status %q, got %q", db.OAuthStatusNeedsReauth, conn.Status)
	}
}

func TestExecuteConnectorAction_OAuthPath_NonExpiredTokenSkipsRefresh(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testgoogle"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testgoogle.send_email", "Send Email")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", nil)

	v := vault.NewMockVaultStore()
	accessVaultID, _ := v.CreateSecret(t.Context(), tx, "access", []byte("fresh-token"))
	// Token expires well in the future (beyond the 5-minute buffer).
	farFuture := time.Now().Add(2 * time.Hour)
	connOAuthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, "google", testhelper.OAuthConnectionOpts{
		AccessTokenVaultID: accessVaultID,
		Scopes:             []string{},
		TokenExpiry:        &farFuture,
		Status:             "active",
	})

	var capturedCreds connectors.Credentials
	reg := connectors.NewRegistry()
	reg.Register(&credCapturingConnector{
		id:      connID,
		actions: []string{"testgoogle.send_email"},
		onExec:  func(creds connectors.Credentials) { capturedCreds = creds },
	})

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID: "google", AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL: "https://oauth2.googleapis.com/token",
		ClientID: "id", ClientSecret: "secret", Source: oauth.SourceBuiltIn,
	})

	deps := &Deps{DB: tx, Vault: v, Connectors: reg, OAuthProviders: oauthReg}

	result, err := executeConnectorAction(t.Context(), deps, uid, "testgoogle.send_email", json.RawMessage(`{}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	// Token should be the original, un-refreshed token.
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
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	connID := "testgoogle"
	testhelper.InsertConnector(t, tx, connID)
	testhelper.InsertConnectorAction(t, tx, connID, "testgoogle.send_email", "Send Email")
	testhelper.InsertConnectorRequiredCredentialOAuth(t, tx, connID, "google", "google", nil)

	v := vault.NewMockVaultStore()
	accessVaultID, _ := v.CreateSecret(t.Context(), tx, "access", []byte("no-expiry-token"))
	connOAuthID := testhelper.GenerateID(t, "oconn_")
	testhelper.InsertOAuthConnectionFull(t, tx, connOAuthID, uid, "google", testhelper.OAuthConnectionOpts{
		AccessTokenVaultID: accessVaultID,
		Scopes:             []string{},
		TokenExpiry:        nil, // no expiry info
		Status:             "active",
	})

	var capturedCreds connectors.Credentials
	reg := connectors.NewRegistry()
	reg.Register(&credCapturingConnector{
		id:      connID,
		actions: []string{"testgoogle.send_email"},
		onExec:  func(creds connectors.Credentials) { capturedCreds = creds },
	})

	oauthReg := oauth.NewRegistry()
	_ = oauthReg.Register(oauth.Provider{
		ID: "google", AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL: "https://oauth2.googleapis.com/token",
		ClientID: "id", ClientSecret: "secret", Source: oauth.SourceBuiltIn,
	})

	deps := &Deps{DB: tx, Vault: v, Connectors: reg, OAuthProviders: oauthReg}

	result, err := executeConnectorAction(t.Context(), deps, uid, "testgoogle.send_email", json.RawMessage(`{}`))
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

// ── test helpers ────────────────────────────────────────────────────────────

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
