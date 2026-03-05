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

	state, err := createOAuthState(deps, "user-123", "google")
	if err != nil {
		t.Fatalf("createOAuthState: %v", err)
	}
	if state == "" {
		t.Fatal("expected non-empty state")
	}

	userID, provider, err := verifyOAuthState(deps, state)
	if err != nil {
		t.Fatalf("verifyOAuthState: %v", err)
	}
	if userID != "user-123" {
		t.Errorf("expected user-123, got %s", userID)
	}
	if provider != "google" {
		t.Errorf("expected google, got %s", provider)
	}
}

func TestVerifyOAuthState_InvalidSignature(t *testing.T) {
	t.Parallel()
	deps := &Deps{OAuthStateSecret: testOAuthStateSecret}

	state, err := createOAuthState(deps, "user-123", "google")
	if err != nil {
		t.Fatalf("createOAuthState: %v", err)
	}

	// Use a different secret to verify
	otherDeps := &Deps{OAuthStateSecret: "different-secret-at-least-32-chars-long"}
	_, _, err = verifyOAuthState(otherDeps, state)
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

	_, _, err = verifyOAuthState(deps, state)
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

	_, _, err = verifyOAuthState(deps, state)
	if err == nil {
		t.Fatal("expected error for missing claims")
	}
}

func TestVerifyOAuthState_NoSecret(t *testing.T) {
	t.Parallel()
	deps := &Deps{}
	_, err := createOAuthState(deps, "user-123", "google")
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

	// Insert an OAuth connection directly via DB
	v := vault.NewMockVaultStore()
	accessID, err := v.CreateSecret(t.Context(), tx, "test_access", []byte("access-token-value"))
	if err != nil {
		t.Fatalf("vault create access: %v", err)
	}
	refreshID, err := v.CreateSecret(t.Context(), tx, "test_refresh", []byte("refresh-token-value"))
	if err != nil {
		t.Fatalf("vault create refresh: %v", err)
	}

	connID := testhelper.GenerateID(t, "oconn_")
	_, err = db.CreateOAuthConnection(t.Context(), tx, db.CreateOAuthConnectionParams{
		ID:                  connID,
		UserID:              uid,
		Provider:            "google",
		AccessTokenVaultID:  accessID,
		RefreshTokenVaultID: &refreshID,
		Scopes:              []string{"openid", "email"},
	})
	if err != nil {
		t.Fatalf("create oauth connection: %v", err)
	}

	deps := &Deps{
		DB:                tx,
		Vault:             v,
		SupabaseJWTSecret: testJWTSecret,
		OAuthProviders:    oauth.NewRegistry(),
		OAuthStateSecret:  testOAuthStateSecret,
		BaseURL:           "http://localhost:3000",
	}
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
	accessID, _ := v.CreateSecret(t.Context(), tx, "test_access", []byte("token"))
	connID := testhelper.GenerateID(t, "oconn_")
	_, err := db.CreateOAuthConnection(t.Context(), tx, db.CreateOAuthConnectionParams{
		ID:                 connID,
		UserID:             uid1,
		Provider:           "google",
		AccessTokenVaultID: accessID,
		Scopes:             []string{"openid"},
	})
	if err != nil {
		t.Fatalf("create oauth connection: %v", err)
	}

	deps := &Deps{
		DB:                tx,
		Vault:             v,
		SupabaseJWTSecret: testJWTSecret,
		OAuthProviders:    oauth.NewRegistry(),
		OAuthStateSecret:  testOAuthStateSecret,
		BaseURL:           "http://localhost:3000",
	}
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
	accessID, _ := v.CreateSecret(t.Context(), tx, "test_access", []byte("access-token"))
	refreshID, _ := v.CreateSecret(t.Context(), tx, "test_refresh", []byte("refresh-token"))

	connID := testhelper.GenerateID(t, "oconn_")
	_, err := db.CreateOAuthConnection(t.Context(), tx, db.CreateOAuthConnectionParams{
		ID:                  connID,
		UserID:              uid,
		Provider:            "google",
		AccessTokenVaultID:  accessID,
		RefreshTokenVaultID: &refreshID,
		Scopes:              []string{"openid"},
	})
	if err != nil {
		t.Fatalf("create oauth connection: %v", err)
	}

	deps := &Deps{
		DB:                tx,
		Vault:             v,
		SupabaseJWTSecret: testJWTSecret,
		OAuthProviders:    oauth.NewRegistry(),
		OAuthStateSecret:  testOAuthStateSecret,
		BaseURL:           "http://localhost:3000",
	}
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
	accessID, _ := v.CreateSecret(t.Context(), tx, "test_access", []byte("token"))
	connID := testhelper.GenerateID(t, "oconn_")
	_, err := db.CreateOAuthConnection(t.Context(), tx, db.CreateOAuthConnectionParams{
		ID:                 connID,
		UserID:             uid1,
		Provider:           "google",
		AccessTokenVaultID: accessID,
		Scopes:             []string{"openid"},
	})
	if err != nil {
		t.Fatalf("create oauth connection: %v", err)
	}

	deps := &Deps{
		DB:                tx,
		Vault:             v,
		SupabaseJWTSecret: testJWTSecret,
		OAuthProviders:    oauth.NewRegistry(),
		OAuthStateSecret:  testOAuthStateSecret,
		BaseURL:           "http://localhost:3000",
	}
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
	state, err := createOAuthState(deps, uid, "microsoft")
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
	state, err := createOAuthState(deps, uid1, "google")
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

	state, err := createOAuthState(deps, uid, "google")
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

	err := storeOAuthTokens(t.Context(), deps, uid, "google", []string{"openid"}, token)
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
	if err := storeOAuthTokens(t.Context(), deps, uid, "google", []string{"openid"}, token1); err != nil {
		t.Fatalf("first storeOAuthTokens: %v", err)
	}

	// Re-auth with new tokens
	token2 := &oauth2.Token{
		AccessToken:  "access-new",
		RefreshToken: "refresh-new",
		Expiry:       time.Now().Add(2 * time.Hour),
		TokenType:    "Bearer",
	}
	if err := storeOAuthTokens(t.Context(), deps, uid, "google", []string{"openid", "email"}, token2); err != nil {
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
