package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/supersuit-tech/permission-slip-web/db"
	"github.com/supersuit-tech/permission-slip-web/db/testhelper"
	"github.com/supersuit-tech/permission-slip-web/oauth"
	"github.com/supersuit-tech/permission-slip-web/vault"
	"golang.org/x/oauth2"
)

const testOAuthStateSecret = "test-oauth-state-secret-at-least-32-chars"

// insertTestOAuthConnection creates an OAuth connection in the test DB with
// vault-stored tokens. Returns the connection ID for further assertions.
func insertTestOAuthConnection(t *testing.T, tx db.DBTX, v *vault.MockVaultStore, userID, provider string, scopes []string, withRefresh bool) string {
	t.Helper()
	accessID, err := v.CreateSecret(t.Context(), tx, "test_access", []byte("access-token"))
	if err != nil {
		t.Fatalf("vault create access: %v", err)
	}
	var refreshVaultID *string
	if withRefresh {
		refreshID, err := v.CreateSecret(t.Context(), tx, "test_refresh", []byte("refresh-token"))
		if err != nil {
			t.Fatalf("vault create refresh: %v", err)
		}
		refreshVaultID = &refreshID
	}
	connID := testhelper.GenerateID(t, "oconn_")
	_, err = db.CreateOAuthConnection(t.Context(), tx, db.CreateOAuthConnectionParams{
		ID:                  connID,
		UserID:              userID,
		Provider:            provider,
		AccessTokenVaultID:  accessID,
		RefreshTokenVaultID: refreshVaultID,
		Scopes:              scopes,
	})
	if err != nil {
		t.Fatalf("create oauth connection: %v", err)
	}
	return connID
}

// oauthDepsWithVault creates deps with a specific mock vault store. This is
// useful for tests that need to insert vault secrets before creating deps.
func oauthDepsWithVault(tx db.DBTX, v *vault.MockVaultStore) *Deps {
	return &Deps{
		DB:                tx,
		Vault:             v,
		SupabaseJWTSecret: testJWTSecret,
		OAuthProviders:    oauth.NewRegistry(),
		OAuthStateSecret:  testOAuthStateSecret,
		BaseURL:           "http://localhost:3000",
	}
}

func oauthDeps(tx db.DBTX) *Deps {
	reg := oauth.NewRegistry()
	_ = reg.Register(oauth.Provider{
		ID:           "google",
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		Scopes:       []string{"openid", "email"},
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Source:       oauth.SourceBuiltIn,
	})
	_ = reg.Register(oauth.Provider{
		ID:           "unconfigured",
		AuthorizeURL: "https://example.com/auth",
		TokenURL:     "https://example.com/token",
		Scopes:       []string{"read"},
		Source:       oauth.SourceManifest,
		// No client credentials
	})
	return &Deps{
		DB:                tx,
		Vault:             vault.NewMockVaultStore(),
		SupabaseJWTSecret: testJWTSecret,
		OAuthProviders:    reg,
		OAuthStateSecret:  testOAuthStateSecret,
		BaseURL:           "http://localhost:3000",
	}
}

// ── GET /v1/oauth/providers ────────────────────────────────────────────────────

func TestListOAuthProviders_ReturnsRegistered(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/providers", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp oauthProviderListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(resp.Providers))
	}

	// Find the configured provider
	var found bool
	for _, p := range resp.Providers {
		if p.ID == "google" {
			found = true
			if !p.HasCredentials {
				t.Error("expected google to have credentials")
			}
			if p.Source != "built_in" {
				t.Errorf("expected built_in source, got %s", p.Source)
			}
		}
		if p.ID == "unconfigured" {
			if p.HasCredentials {
				t.Error("expected unconfigured provider to NOT have credentials")
			}
		}
	}
	if !found {
		t.Error("expected google provider in list")
	}
}

func TestListOAuthProviders_EmptyWhenNilRegistry(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	deps.OAuthProviders = nil
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/providers", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp oauthProviderListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Providers) != 0 {
		t.Errorf("expected 0 providers, got %d", len(resp.Providers))
	}
}

// ── deduplicateScopes ─────────────────────────────────────────────────────────

func TestDeduplicateScopes(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{"no duplicates", []string{"a", "b", "c"}, []string{"a", "b", "c"}},
		{"with duplicates", []string{"openid", "email", "openid"}, []string{"openid", "email"}},
		{"all same", []string{"x", "x", "x"}, []string{"x"}},
		{"empty", []string{}, []string{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := deduplicateScopes(tt.input)
			if len(got) != len(tt.expected) {
				t.Fatalf("expected %d scopes, got %d: %v", len(tt.expected), len(got), got)
			}
			for i, s := range got {
				if s != tt.expected[i] {
					t.Errorf("index %d: expected %q, got %q", i, tt.expected[i], s)
				}
			}
		})
	}
}

// ── CSRF State Tests ──────────────────────────────────────────────────────────

func TestCreateAndVerifyOAuthState(t *testing.T) {
	t.Parallel()
	deps := &Deps{OAuthStateSecret: testOAuthStateSecret}

	state, err := createOAuthState(deps, "user-123", "google", []string{"openid"}, "")
	if err != nil {
		t.Fatalf("createOAuthState: %v", err)
	}
	if state == "" {
		t.Fatal("expected non-empty state")
	}

	verified, err := verifyOAuthState(deps, state)
	if err != nil {
		t.Fatalf("verifyOAuthState: %v", err)
	}
	if verified.UserID != "user-123" {
		t.Errorf("expected user-123, got %s", verified.UserID)
	}
	if verified.Provider != "google" {
		t.Errorf("expected google, got %s", verified.Provider)
	}
	if len(verified.Scopes) != 1 || verified.Scopes[0] != "openid" {
		t.Errorf("expected [openid], got %v", verified.Scopes)
	}
}

func TestVerifyOAuthState_InvalidSignature(t *testing.T) {
	t.Parallel()
	deps := &Deps{OAuthStateSecret: testOAuthStateSecret}

	state, err := createOAuthState(deps, "user-123", "google", []string{"openid"}, "")
	if err != nil {
		t.Fatalf("createOAuthState: %v", err)
	}

	// Use a different secret to verify
	otherDeps := &Deps{OAuthStateSecret: "different-secret-at-least-32-chars-long"}
	_, err = verifyOAuthState(otherDeps, state)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestVerifyOAuthState_ExpiredToken(t *testing.T) {
	t.Parallel()
	deps := &Deps{OAuthStateSecret: testOAuthStateSecret}

	// Create a token that's already expired
	claims := jwt.MapClaims{
		"sub":      "user-123",
		"provider": "google",
		"iat":      time.Now().Add(-20 * time.Minute).Unix(),
		"exp":      time.Now().Add(-10 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	state, err := token.SignedString([]byte(testOAuthStateSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = verifyOAuthState(deps, state)
	if err == nil {
		t.Fatal("expected error for expired token")
	}
}

func TestVerifyOAuthState_MissingClaims(t *testing.T) {
	t.Parallel()
	deps := &Deps{OAuthStateSecret: testOAuthStateSecret}

	// Create a token with missing sub claim
	claims := jwt.MapClaims{
		"provider": "google",
		"iat":      time.Now().Unix(),
		"exp":      time.Now().Add(10 * time.Minute).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	state, err := token.SignedString([]byte(testOAuthStateSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	_, err = verifyOAuthState(deps, state)
	if err == nil {
		t.Fatal("expected error for missing claims")
	}
}

func TestVerifyOAuthState_NoSecret(t *testing.T) {
	t.Parallel()
	deps := &Deps{}
	_, err := createOAuthState(deps, "user-123", "google", []string{"openid"}, "")
	if err == nil {
		t.Fatal("expected error when no secret configured")
	}
}

// ── GET /v1/oauth/{provider}/authorize ────────────────────────────────────────

func TestOAuthAuthorize_Redirect(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/google/authorize", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}

	location := w.Header().Get("Location")
	if location == "" {
		t.Fatal("expected Location header")
	}
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	if parsed.Host != "accounts.google.com" {
		t.Errorf("expected google host, got %s", parsed.Host)
	}
	if parsed.Query().Get("state") == "" {
		t.Error("expected state param in redirect URL")
	}
	if parsed.Query().Get("client_id") != "test-client-id" {
		t.Errorf("expected test-client-id, got %s", parsed.Query().Get("client_id"))
	}
	if !strings.Contains(parsed.Query().Get("redirect_uri"), "/api/v1/oauth/google/callback") {
		t.Errorf("expected callback URL in redirect_uri, got %s", parsed.Query().Get("redirect_uri"))
	}
}

func TestOAuthAuthorize_ProviderNotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/nonexistent/authorize", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOAuthAuthorize_ProviderUnconfigured(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/unconfigured/authorize", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeErrorResponse(t, w.Body.Bytes())
	if resp.Error.Code != ErrOAuthProviderUnconfigured {
		t.Errorf("expected error code %s, got %s", ErrOAuthProviderUnconfigured, resp.Error.Code)
	}
}

func TestOAuthAuthorize_InvalidProviderID(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	// Provider ID with uppercase and special chars should be rejected
	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/INVALID%21/authorize", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOAuthAuthorize_Unauthenticated(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	deps := oauthDeps(tx)
	router := NewRouter(deps)

	r := httptest.NewRequest(http.MethodGet, "/v1/oauth/google/authorize", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

// ── GET /v1/oauth/connections ─────────────────────────────────────────────────

func TestListOAuthConnections_Empty(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/connections", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp oauthConnectionListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Connections) != 0 {
		t.Errorf("expected 0 connections, got %d", len(resp.Connections))
	}
}

func TestListOAuthConnections_ReturnsUserConnections(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	v := vault.NewMockVaultStore()
	insertTestOAuthConnection(t, tx, v, uid, "google", []string{"openid", "email"}, true)

	deps := oauthDepsWithVault(tx, v)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/connections", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp oauthConnectionListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Connections) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(resp.Connections))
	}
	conn := resp.Connections[0]
	if conn.Provider != "google" {
		t.Errorf("expected google, got %s", conn.Provider)
	}
	if conn.Status != "active" {
		t.Errorf("expected active, got %s", conn.Status)
	}
	if len(conn.Scopes) != 2 {
		t.Errorf("expected 2 scopes, got %d", len(conn.Scopes))
	}
}

func TestListOAuthConnections_IsolatedByUser(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u1_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:8])

	v := vault.NewMockVaultStore()
	insertTestOAuthConnection(t, tx, v, uid1, "google", []string{"openid"}, false)

	deps := oauthDepsWithVault(tx, v)
	router := NewRouter(deps)

	// User 2 should see no connections
	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/connections", uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp oauthConnectionListResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Connections) != 0 {
		t.Errorf("expected 0 connections for user2, got %d", len(resp.Connections))
	}
}

// ── DELETE /v1/oauth/connections/{provider} ───────────────────────────────────

func TestDeleteOAuthConnection_Success(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	v := vault.NewMockVaultStore()
	insertTestOAuthConnection(t, tx, v, uid, "google", []string{"openid"}, true)

	deps := oauthDepsWithVault(tx, v)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/v1/oauth/connections/google", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp oauthDisconnectResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Provider != "google" {
		t.Errorf("expected google, got %s", resp.Provider)
	}

	// Verify the connection is actually gone
	conn, err := db.GetOAuthConnectionByProvider(t.Context(), tx, uid, "google")
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	if conn != nil {
		t.Error("expected connection to be deleted")
	}
}

func TestDeleteOAuthConnection_NotFound(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodDelete, "/v1/oauth/connections/google", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	resp := decodeErrorResponse(t, w.Body.Bytes())
	if resp.Error.Code != ErrOAuthConnectionNotFound {
		t.Errorf("expected error code %s, got %s", ErrOAuthConnectionNotFound, resp.Error.Code)
	}
}

func TestDeleteOAuthConnection_OtherUserCannot(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u1_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:8])

	v := vault.NewMockVaultStore()
	insertTestOAuthConnection(t, tx, v, uid1, "google", []string{"openid"}, false)

	deps := oauthDepsWithVault(tx, v)
	router := NewRouter(deps)

	// User 2 tries to delete user 1's connection
	r := authenticatedRequest(t, http.MethodDelete, "/v1/oauth/connections/google", uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d: %s", w.Code, w.Body.String())
	}

	// Verify user 1's connection still exists
	conn, err := db.GetOAuthConnectionByProvider(t.Context(), tx, uid1, "google")
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	if conn == nil {
		t.Error("expected user1's connection to still exist")
	}
}

// ── GET /v1/oauth/{provider}/callback ─────────────────────────────────────────

func TestOAuthCallback_MissingState(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/google/callback?code=test-code", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	// Should redirect to frontend with error
	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "oauth_status=error") {
		t.Errorf("expected error status in redirect, got: %s", location)
	}
}

func TestOAuthCallback_InvalidState(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/google/callback?code=test-code&state=invalid-jwt", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "oauth_status=error") {
		t.Errorf("expected error status in redirect, got: %s", location)
	}
}

func TestOAuthCallback_ProviderMismatch(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	// Create state for "microsoft" but hit google callback
	state, err := createOAuthState(deps, uid, "microsoft", []string{"openid"}, "")
	if err != nil {
		t.Fatalf("create state: %v", err)
	}

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/google/callback?code=test-code&state="+url.QueryEscape(state), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "oauth_status=error") {
		t.Errorf("expected error status in redirect, got: %s", location)
	}
}

func TestOAuthCallback_UserMismatch(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid1 := testhelper.GenerateUID(t)
	uid2 := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid1, "u1_"+uid1[:8])
	testhelper.InsertUser(t, tx, uid2, "u2_"+uid2[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	// State is for user1 but session is user2
	state, err := createOAuthState(deps, uid1, "google", []string{"openid"}, "")
	if err != nil {
		t.Fatalf("create state: %v", err)
	}

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/google/callback?code=test-code&state="+url.QueryEscape(state), uid2)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "oauth_status=error") {
		t.Errorf("expected error status in redirect, got: %s", location)
	}
}

func TestOAuthCallback_ProviderError(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/google/callback?error=access_denied&error_description=User+denied+consent", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "oauth_status=error") {
		t.Errorf("expected error status in redirect, got: %s", location)
	}
	if !strings.Contains(location, "User+denied+consent") && !strings.Contains(location, "User%20denied%20consent") {
		t.Errorf("expected error description in redirect, got: %s", location)
	}
}

func TestOAuthCallback_MissingCode(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)
	router := NewRouter(deps)

	state, err := createOAuthState(deps, uid, "google", []string{"openid"}, "")
	if err != nil {
		t.Fatalf("create state: %v", err)
	}

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/google/callback?state="+url.QueryEscape(state), uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "oauth_status=error") {
		t.Errorf("expected error status in redirect, got: %s", location)
	}
}

// ── URL helpers ───────────────────────────────────────────────────────────────

func TestOAuthCallbackURL(t *testing.T) {
	t.Parallel()
	deps := &Deps{BaseURL: "https://app.permissionslip.dev"}
	u := oauthCallbackURL(deps, "google")
	if u != "https://app.permissionslip.dev/api/v1/oauth/google/callback" {
		t.Errorf("unexpected callback URL: %s", u)
	}

	// OAuthRedirectBaseURL takes precedence
	deps.OAuthRedirectBaseURL = "https://custom.example.com"
	u = oauthCallbackURL(deps, "microsoft")
	if u != "https://custom.example.com/api/v1/oauth/microsoft/callback" {
		t.Errorf("unexpected callback URL with override: %s", u)
	}
}

// ── storeOAuthTokens (integration) ───────────────────────────────────────────

func TestStoreOAuthTokens_CreateNew(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)

	token := &oauth2.Token{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		Expiry:       time.Now().Add(time.Hour),
		TokenType:    "Bearer",
	}

	err := storeOAuthTokens(t.Context(), deps, uid, "google", []string{"openid"}, token, nil)
	if err != nil {
		t.Fatalf("storeOAuthTokens: %v", err)
	}

	// Verify the connection was created
	conn, err := db.GetOAuthConnectionByProvider(t.Context(), tx, uid, "google")
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	if conn == nil {
		t.Fatal("expected connection to exist")
	}
	if conn.Provider != "google" {
		t.Errorf("expected google, got %s", conn.Provider)
	}
	if conn.Status != db.OAuthStatusActive {
		t.Errorf("expected active, got %s", conn.Status)
	}
	if conn.RefreshTokenVaultID == nil {
		t.Error("expected refresh token vault ID to be set")
	}
}

func TestStoreOAuthTokens_ReplacesExisting(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)

	// Create first connection
	token1 := &oauth2.Token{
		AccessToken:  "access-old",
		RefreshToken: "refresh-old",
		Expiry:       time.Now().Add(time.Hour),
		TokenType:    "Bearer",
	}
	if err := storeOAuthTokens(t.Context(), deps, uid, "google", []string{"openid"}, token1, nil); err != nil {
		t.Fatalf("first storeOAuthTokens: %v", err)
	}

	// Re-auth with new tokens
	token2 := &oauth2.Token{
		AccessToken:  "access-new",
		RefreshToken: "refresh-new",
		Expiry:       time.Now().Add(2 * time.Hour),
		TokenType:    "Bearer",
	}
	if err := storeOAuthTokens(t.Context(), deps, uid, "google", []string{"openid", "email"}, token2, nil); err != nil {
		t.Fatalf("second storeOAuthTokens: %v", err)
	}

	// Should still be exactly one connection
	conns, err := db.ListOAuthConnectionsByUser(t.Context(), tx, uid)
	if err != nil {
		t.Fatalf("list connections: %v", err)
	}
	if len(conns) != 1 {
		t.Fatalf("expected 1 connection, got %d", len(conns))
	}
	if len(conns[0].Scopes) != 2 {
		t.Errorf("expected 2 scopes after re-auth, got %d", len(conns[0].Scopes))
	}
}

// ── Shopify per-shop URL helpers ──────────────────────────────────────────────

func TestValidateShopSubdomain(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{"bare subdomain", "mystore", "mystore", false},
		{"full domain", "mystore.myshopify.com", "mystore", false},
		{"uppercase normalized", "MyStore", "mystore", false},
		{"with hyphens", "my-cool-store", "my-cool-store", false},
		{"trailing slash", "mystore/", "mystore", false},
		{"whitespace", "  mystore  ", "mystore", false},
		{"full domain uppercase", "MyStore.myshopify.com", "mystore", false},
		{"empty", "", "", true},
		{"only whitespace", "   ", "", true},
		{"invalid domain", "mystore.example.com", "", true},
		{"starts with hyphen", "-mystore", "", true},
		{"ends with hyphen", "mystore-", "", true},
		{"special chars", "my_store!", "", true},
		{"just .myshopify.com", ".myshopify.com", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := validateShopSubdomain(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("validateShopSubdomain(%q): err=%v, wantErr=%v", tt.input, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("validateShopSubdomain(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestProviderNeedsShop(t *testing.T) {
	t.Parallel()
	shopProvider := oauth.Provider{
		AuthorizeURL: "https://{shop}.myshopify.com/admin/oauth/authorize",
		TokenURL:     "https://{shop}.myshopify.com/admin/oauth/access_token",
	}
	if !providerNeedsShop(shopProvider) {
		t.Error("expected providerNeedsShop to return true for Shopify-style URLs")
	}

	staticProvider := oauth.Provider{
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
	}
	if providerNeedsShop(staticProvider) {
		t.Error("expected providerNeedsShop to return false for static URLs")
	}
}

func TestResolveShopURLs(t *testing.T) {
	t.Parallel()
	p := oauth.Provider{
		ID:           "shopify",
		AuthorizeURL: "https://{shop}.myshopify.com/admin/oauth/authorize",
		TokenURL:     "https://{shop}.myshopify.com/admin/oauth/access_token",
	}
	resolved := resolveShopURLs(p, "mystore")
	if resolved.AuthorizeURL != "https://mystore.myshopify.com/admin/oauth/authorize" {
		t.Errorf("unexpected AuthorizeURL: %s", resolved.AuthorizeURL)
	}
	if resolved.TokenURL != "https://mystore.myshopify.com/admin/oauth/access_token" {
		t.Errorf("unexpected TokenURL: %s", resolved.TokenURL)
	}
	// Original should be unchanged.
	if p.AuthorizeURL != "https://{shop}.myshopify.com/admin/oauth/authorize" {
		t.Error("resolveShopURLs mutated the original provider")
	}
}

func TestCreateAndVerifyOAuthState_WithShop(t *testing.T) {
	t.Parallel()
	deps := &Deps{OAuthStateSecret: testOAuthStateSecret}

	state, err := createOAuthState(deps, "user-456", "shopify", []string{"write_orders"}, "mystore")
	if err != nil {
		t.Fatalf("createOAuthState: %v", err)
	}

	verified, err := verifyOAuthState(deps, state)
	if err != nil {
		t.Fatalf("verifyOAuthState: %v", err)
	}
	if verified.UserID != "user-456" {
		t.Errorf("expected user-456, got %s", verified.UserID)
	}
	if verified.Provider != "shopify" {
		t.Errorf("expected shopify, got %s", verified.Provider)
	}
	if verified.Shop != "mystore" {
		t.Errorf("expected shop=mystore, got %q", verified.Shop)
	}
	if len(verified.Scopes) != 1 || verified.Scopes[0] != "write_orders" {
		t.Errorf("expected [write_orders], got %v", verified.Scopes)
	}
}

func TestCreateAndVerifyOAuthState_EmptyShop(t *testing.T) {
	t.Parallel()
	deps := &Deps{OAuthStateSecret: testOAuthStateSecret}

	state, err := createOAuthState(deps, "user-123", "google", []string{"openid"}, "")
	if err != nil {
		t.Fatalf("createOAuthState: %v", err)
	}

	verified, err := verifyOAuthState(deps, state)
	if err != nil {
		t.Fatalf("verifyOAuthState: %v", err)
	}
	if verified.Shop != "" {
		t.Errorf("expected empty shop, got %q", verified.Shop)
	}
}

// ── Shopify OAuth authorize ───────────────────────────────────────────────────

func oauthDepsWithShopify(tx db.DBTX) *Deps {
	reg := oauth.NewRegistry()
	_ = reg.Register(oauth.Provider{
		ID:           "shopify",
		AuthorizeURL: "https://{shop}.myshopify.com/admin/oauth/authorize",
		TokenURL:     "https://{shop}.myshopify.com/admin/oauth/access_token",
		Scopes:       []string{"write_orders", "write_products"},
		ClientID:     "shopify-client-id",
		ClientSecret: "shopify-client-secret",
		Source:       oauth.SourceBuiltIn,
	})
	_ = reg.Register(oauth.Provider{
		ID:           "google",
		AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:     "https://oauth2.googleapis.com/token",
		Scopes:       []string{"openid"},
		ClientID:     "google-client-id",
		ClientSecret: "google-client-secret",
		Source:       oauth.SourceBuiltIn,
	})
	return &Deps{
		DB:                tx,
		Vault:             vault.NewMockVaultStore(),
		SupabaseJWTSecret: testJWTSecret,
		OAuthProviders:    reg,
		OAuthStateSecret:  testOAuthStateSecret,
		BaseURL:           "http://localhost:3000",
	}
}

func TestOAuthAuthorize_Shopify_RequiresShopParam(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDepsWithShopify(tx)
	router := NewRouter(deps)

	// No shop param → 400
	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/shopify/authorize", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOAuthAuthorize_Shopify_InvalidShop(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDepsWithShopify(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/shopify/authorize?shop=-invalid", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}

func TestOAuthAuthorize_Shopify_ValidShop(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDepsWithShopify(tx)
	router := NewRouter(deps)

	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/shopify/authorize?shop=mystore", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}

	location := w.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	// Should redirect to mystore.myshopify.com (resolved template)
	if parsed.Host != "mystore.myshopify.com" {
		t.Errorf("expected mystore.myshopify.com host, got %s", parsed.Host)
	}
	if !strings.Contains(parsed.Path, "/admin/oauth/authorize") {
		t.Errorf("expected Shopify authorize path, got %s", parsed.Path)
	}
	if parsed.Query().Get("client_id") != "shopify-client-id" {
		t.Errorf("expected shopify-client-id, got %s", parsed.Query().Get("client_id"))
	}
	if parsed.Query().Get("state") == "" {
		t.Error("expected state param in redirect URL")
	}
}

func TestOAuthAuthorize_Shopify_FullDomainShop(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDepsWithShopify(tx)
	router := NewRouter(deps)

	// Full domain form should also work
	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/shopify/authorize?shop=mystore.myshopify.com", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}

	location := w.Header().Get("Location")
	parsed, err := url.Parse(location)
	if err != nil {
		t.Fatalf("parse location: %v", err)
	}
	if parsed.Host != "mystore.myshopify.com" {
		t.Errorf("expected mystore.myshopify.com host, got %s", parsed.Host)
	}
}

func TestOAuthAuthorize_NonShopProvider_IgnoresShopParam(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDepsWithShopify(tx)
	router := NewRouter(deps)

	// Google provider should work without shop param and ignore it
	r := authenticatedRequest(t, http.MethodGet, "/v1/oauth/google/authorize", uid)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}
}

// ── storeOAuthTokens with stateExtra ──────────────────────────────────────────

func TestStoreOAuthTokens_WithStateExtra(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	deps := oauthDeps(tx)

	token := &oauth2.Token{
		AccessToken: "shopify-access-token",
		TokenType:   "Bearer",
		// Shopify tokens don't expire
	}

	stateExtra := map[string]string{"shop_domain": "mystore.myshopify.com"}
	err := storeOAuthTokens(t.Context(), deps, uid, "shopify", []string{"write_orders"}, token, stateExtra)
	if err != nil {
		t.Fatalf("storeOAuthTokens: %v", err)
	}

	conn, err := db.GetOAuthConnectionByProvider(t.Context(), tx, uid, "shopify")
	if err != nil {
		t.Fatalf("get connection: %v", err)
	}
	if conn == nil {
		t.Fatal("expected connection to exist")
	}
	if conn.RefreshTokenVaultID != nil {
		t.Error("expected no refresh token for Shopify")
	}

	// Verify extra_data contains shop_domain
	if len(conn.ExtraData) == 0 {
		t.Fatal("expected extra_data to be set")
	}
	var extra map[string]string
	if err := json.Unmarshal(conn.ExtraData, &extra); err != nil {
		t.Fatalf("unmarshal extra_data: %v", err)
	}
	if extra["shop_domain"] != "mystore.myshopify.com" {
		t.Errorf("expected shop_domain=mystore.myshopify.com, got %q", extra["shop_domain"])
	}
}

// ── extractTokenExtraData with stateExtra ─────────────────────────────────────

func TestExtractTokenExtraData_MergesStateExtra(t *testing.T) {
	t.Parallel()
	token := &oauth2.Token{
		AccessToken: "test",
		TokenType:   "Bearer",
	}

	stateExtra := map[string]string{"shop_domain": "mystore.myshopify.com"}
	raw := extractTokenExtraData(token, stateExtra)
	if raw == nil {
		t.Fatal("expected non-nil extra data")
	}

	var extra map[string]string
	if err := json.Unmarshal(raw, &extra); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if extra["shop_domain"] != "mystore.myshopify.com" {
		t.Errorf("expected shop_domain=mystore.myshopify.com, got %q", extra["shop_domain"])
	}
}

func TestExtractTokenExtraData_NilStateExtra(t *testing.T) {
	t.Parallel()
	token := &oauth2.Token{
		AccessToken: "test",
		TokenType:   "Bearer",
	}

	raw := extractTokenExtraData(token, nil)
	// No token extras, no state extras → nil
	if raw != nil {
		t.Errorf("expected nil extra data, got %s", string(raw))
	}
}

func docuSignTestServer(t *testing.T, accounts []map[string]any) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") == "" {
			http.Error(w, "missing authorization", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"accounts": accounts})
	}))
}

func TestFetchDocuSignUserInfo_DefaultAccount(t *testing.T) {
	t.Parallel()
	server := docuSignTestServer(t, []map[string]any{
		{"account_id": "acc-non-default", "is_default": false, "base_uri": "https://na1.docusign.net"},
		{"account_id": "acc-default-456", "is_default": true, "base_uri": "https://na2.docusign.net"},
	})
	defer server.Close()

	got, err := fetchDocuSignUserInfo(t.Context(), "test-token", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["account_id"] != "acc-default-456" {
		t.Errorf("account_id = %q, want %q", got["account_id"], "acc-default-456")
	}
	wantBaseURL := "https://na2.docusign.net/restapi/v2.1"
	if got["base_url"] != wantBaseURL {
		t.Errorf("base_url = %q, want %q", got["base_url"], wantBaseURL)
	}
}

func TestFetchDocuSignUserInfo_FallsBackToFirstAccount(t *testing.T) {
	t.Parallel()
	// No account is marked as default → should use the first one.
	server := docuSignTestServer(t, []map[string]any{
		{"account_id": "acc-first", "is_default": false, "base_uri": "https://na1.docusign.net"},
		{"account_id": "acc-second", "is_default": false, "base_uri": "https://na2.docusign.net"},
	})
	defer server.Close()

	got, err := fetchDocuSignUserInfo(t.Context(), "test-token", server.URL)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got["account_id"] != "acc-first" {
		t.Errorf("account_id = %q, want %q", got["account_id"], "acc-first")
	}
}

func TestFetchDocuSignUserInfo_NoAccounts(t *testing.T) {
	t.Parallel()
	server := docuSignTestServer(t, []map[string]any{})
	defer server.Close()

	_, err := fetchDocuSignUserInfo(t.Context(), "test-token", server.URL)
	if err == nil {
		t.Fatal("expected error for empty accounts list")
	}
}

func TestFetchDocuSignUserInfo_HTTPError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	_, err := fetchDocuSignUserInfo(t.Context(), "test-token", server.URL)
	if err == nil {
		t.Fatal("expected error for HTTP 500 response")
	}
}

func TestFetchDocuSignUserInfo_InvalidBaseURI(t *testing.T) {
	t.Parallel()
	// base_uri doesn't end with .docusign.net — should be rejected for SSRF protection.
	server := docuSignTestServer(t, []map[string]any{
		{"account_id": "acc-123", "is_default": true, "base_uri": "https://evil.example.com"},
	})
	defer server.Close()

	_, err := fetchDocuSignUserInfo(t.Context(), "test-token", server.URL)
	if err == nil {
		t.Fatal("expected error for non-DocuSign base_uri")
	}
}

func TestIsURLExtraKey_IncludesBaseURL(t *testing.T) {
	t.Parallel()
	cases := []struct {
		key  string
		want bool
	}{
		{"base_url", true},
		{"instance_url", true},
		{"shop_domain", false},
		{"account_id", false},
	}
	for _, tc := range cases {
		if got := isURLExtraKey(tc.key); got != tc.want {
			t.Errorf("isURLExtraKey(%q) = %v, want %v", tc.key, got, tc.want)
		}
	}
}
