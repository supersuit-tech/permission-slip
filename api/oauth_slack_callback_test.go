package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/db"
	"github.com/supersuit-tech/permission-slip/db/testhelper"
	"github.com/supersuit-tech/permission-slip/oauth"
	"github.com/supersuit-tech/permission-slip/vault"
	"golang.org/x/oauth2"
)

// slackCallbackDeps registers a "slack" provider pointing at a fake token
// server. It mirrors oauthDepsWithSlack but lets each test control the token
// endpoint response.
func slackCallbackDeps(tx db.DBTX, tokenURL string) *Deps {
	reg := oauth.NewRegistry()
	_ = reg.Register(oauth.Provider{
		ID:           "slack",
		AuthorizeURL: "https://slack.com/oauth/v2_user/authorize",
		TokenURL:     tokenURL,
		Scopes:       []string{"channels:read"},
		AuthorizeParams: map[string]string{
			"scope": "channels:read",
		},
		AuthStyle:    oauth2.AuthStyleInParams,
		ClientID:     "test-slack-client-id",
		ClientSecret: "test-slack-client-secret",
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

// TestOAuthCallback_SlackV2UserEndpointSucceeds verifies that when the Slack
// token endpoint returns a v2_user-shaped response (top-level access_token),
// the callback completes the flow and redirects with oauth_status=success.
//
// This locks in the contract our production config relies on: oauth.v2.user.access
// returns the user token at the top level, so Go's oauth2 library accepts it.
func TestOAuthCallback_SlackV2UserEndpointSucceeds(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Fake oauth.v2.user.access: user token at the top level.
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"access_token": "xoxp-user-token",
			"token_type": "user",
			"scope": "channels:read",
			"team": {"id": "T123", "name": "Acme Corp"},
			"authed_user": {"id": "U123", "scope": "channels:read"}
		}`))
	}))
	defer tokenSrv.Close()

	deps := slackCallbackDeps(tx, tokenSrv.URL)
	state, err := createOAuthState(deps, uid, "slack", []string{"channels:read"}, "", "", "", "")
	if err != nil {
		t.Fatalf("create state: %v", err)
	}

	router := NewRouter(deps)
	r := httptest.NewRequest(http.MethodGet,
		"/oauth/slack/callback?code=test-code&state="+url.QueryEscape(state), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "oauth_status=success") {
		t.Errorf("expected success status in redirect, got: %s", location)
	}
	if !strings.Contains(location, "oauth_connection_id=") {
		t.Errorf("expected connection id in redirect, got: %s", location)
	}
}

// TestOAuthCallback_SlackV2AccessNestedResponseFails is the regression guard
// for Sentry PERMISSION-SLIP-GO-N. If someone points Slack at the standard
// oauth.v2.access endpoint (or a compatible mock), the response omits the
// top-level access_token when only user scopes are requested — and Go's
// oauth2 library rejects it at cfg.Exchange with "server response missing
// access_token" BEFORE our NormalizeSlackUserOAuthToken can run.
//
// This test documents that failure mode so anyone swapping endpoints sees
// exactly why it breaks. Do not "fix" this test by adding a custom exchange
// or by reverting to oauth.v2.access — use the v2_user endpoints instead.
func TestOAuthCallback_SlackV2AccessNestedResponseFails(t *testing.T) {
	t.Parallel()
	tx := testhelper.SetupTestDB(t)
	uid := testhelper.GenerateUID(t)
	testhelper.InsertUser(t, tx, uid, "u_"+uid[:8])

	// Fake oauth.v2.access response: empty top-level access_token, user token
	// nested under authed_user. This is what Slack returns from oauth.v2.access
	// when only user scopes are requested.
	tokenSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"ok": true,
			"access_token": "",
			"token_type": "",
			"scope": "",
			"team": {"id": "T123", "name": "Acme Corp"},
			"authed_user": {
				"id": "U123",
				"scope": "channels:read",
				"access_token": "xoxp-user-token",
				"token_type": "user"
			}
		}`))
	}))
	defer tokenSrv.Close()

	deps := slackCallbackDeps(tx, tokenSrv.URL)
	state, err := createOAuthState(deps, uid, "slack", []string{"channels:read"}, "", "", "", "")
	if err != nil {
		t.Fatalf("create state: %v", err)
	}

	router := NewRouter(deps)
	r := httptest.NewRequest(http.MethodGet,
		"/oauth/slack/callback?code=test-code&state="+url.QueryEscape(state), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r)

	if w.Code != http.StatusTemporaryRedirect {
		t.Fatalf("expected 307, got %d: %s", w.Code, w.Body.String())
	}
	location := w.Header().Get("Location")
	if !strings.Contains(location, "oauth_status=error") {
		t.Errorf("expected error status in redirect (nested-only response should fail), got: %s", location)
	}
	if !strings.Contains(location, "Token+exchange+failed") && !strings.Contains(location, "Token%20exchange%20failed") {
		t.Errorf("expected 'Token exchange failed' in redirect, got: %s", location)
	}
}
