package jira

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Compile-time interface checks.
var (
	_ connectors.Connector        = (*JiraConnector)(nil)
	_ connectors.ManifestProvider = (*JiraConnector)(nil)
)

func TestJiraConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "jira" {
		t.Errorf("ID() = %q, want %q", c.ID(), "jira")
	}
}

func TestJiraConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid basic auth credentials",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "valid oauth credentials",
			creds:   validOAuthCreds(),
			wantErr: false,
		},
		{
			name:    "missing site (basic auth)",
			creds:   connectors.NewCredentials(map[string]string{"email": "user@example.com", "api_token": "tok"}),
			wantErr: true,
			errMsg:  "site",
		},
		{
			name:    "missing email (basic auth)",
			creds:   connectors.NewCredentials(map[string]string{"site": "mysite", "api_token": "tok"}),
			wantErr: true,
			errMsg:  "email",
		},
		{
			name:    "missing api_token (basic auth)",
			creds:   connectors.NewCredentials(map[string]string{"site": "mysite", "email": "user@example.com"}),
			wantErr: true,
			errMsg:  "api_token",
		},
		{
			name:    "empty site (basic auth)",
			creds:   connectors.NewCredentials(map[string]string{"site": "", "email": "user@example.com", "api_token": "tok"}),
			wantErr: true,
			errMsg:  "site",
		},
		{
			name:    "empty access_token falls back to basic auth validation",
			creds:   connectors.NewCredentials(map[string]string{"access_token": ""}),
			wantErr: true,
		},
	}

	c := New()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(context.Background(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestJiraConnector_Do_BasicAuth(t *testing.T) {
	t.Parallel()

	wantAuth := "Basic " + base64.StdEncoding.EncodeToString([]byte("user@example.com:test-api-token-123"))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Errorf("Authorization = %q, want %q", got, wantAuth)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want %q", got, "application/json")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", nil, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("response status = %q, want %q", resp["status"], "ok")
	}
}

func TestJiraConnector_Do_PostBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"id": "12345"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	body := map[string]string{"summary": "Test issue"}
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodPost, "/issue", body, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["id"] != "12345" {
		t.Errorf("id = %q, want %q", resp["id"], "12345")
	}
}

func TestJiraConnector_Do_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(ctx, validCreds(), http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestJiraConnector_Do_MissingCredentials(t *testing.T) {
	t.Parallel()

	conn := newForTest(nil, "http://localhost")
	tests := []struct {
		name  string
		creds connectors.Credentials
	}{
		{
			name:  "missing email",
			creds: connectors.NewCredentials(map[string]string{"site": "s", "api_token": "t"}),
		},
		{
			name:  "missing api_token",
			creds: connectors.NewCredentials(map[string]string{"site": "s", "email": "e"}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := conn.do(t.Context(), tt.creds, http.MethodGet, "/test", nil, nil)
			if err == nil {
				t.Fatal("do() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestJiraConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"jira.create_issue",
		"jira.update_issue",
		"jira.transition_issue",
		"jira.add_comment",
		"jira.assign_issue",
		"jira.search",
	}
	for _, at := range want {
		if _, ok := actions[at]; !ok {
			t.Errorf("Actions() missing %q", at)
		}
	}
	if len(actions) != len(want) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(want))
	}
}

func TestJiraConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "jira" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "jira")
	}
	if m.Name != "Jira" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Jira")
	}
	if len(m.Actions) != 6 {
		t.Fatalf("Manifest().Actions has %d items, want 6", len(m.Actions))
	}

	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"jira.create_issue", "jira.update_issue", "jira.transition_issue",
		"jira.add_comment", "jira.assign_issue", "jira.search",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}

	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 2", len(m.RequiredCredentials))
	}

	// First credential should be OAuth (default/recommended).
	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "jira" {
		t.Errorf("oauth credential service = %q, want %q", oauthCred.Service, "jira")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("oauth credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "atlassian" {
		t.Errorf("oauth credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "atlassian")
	}
	if len(oauthCred.OAuthScopes) == 0 {
		t.Error("oauth credential oauth_scopes is empty, want scopes")
	}

	// Second credential should be basic auth (alternative).
	basicCred := m.RequiredCredentials[1]
	if basicCred.Service != "jira" {
		t.Errorf("basic credential service = %q, want %q", basicCred.Service, "jira")
	}
	if basicCred.AuthType != "basic" {
		t.Errorf("basic credential auth_type = %q, want %q", basicCred.AuthType, "basic")
	}
	if basicCred.InstructionsURL == "" {
		t.Error("basic credential instructions_url is empty, want a URL")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestJiraConnector_ActionsMatchManifest(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	manifest := c.Manifest()

	manifestTypes := make(map[string]bool, len(manifest.Actions))
	for _, a := range manifest.Actions {
		manifestTypes[a.ActionType] = true
	}

	for actionType := range actions {
		if !manifestTypes[actionType] {
			t.Errorf("Actions() has %q but Manifest() does not", actionType)
		}
	}
	for _, a := range manifest.Actions {
		if _, ok := actions[a.ActionType]; !ok {
			t.Errorf("Manifest() has %q but Actions() does not", a.ActionType)
		}
	}
}

func TestJiraConnector_Do_OAuthBearerToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		wantAuth := "Bearer test-oauth-token-456"
		if got := r.Header.Get("Authorization"); got != wantAuth {
			t.Errorf("Authorization = %q, want %q", got, wantAuth)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validOAuthCreds(), http.MethodGet, "/test", nil, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("response status = %q, want %q", resp["status"], "ok")
	}
}

func TestJiraConnector_OAuthAPIBase_DiscoverCloudID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer test-oauth-token-456" {
			t.Errorf("accessible-resources auth = %q, want Bearer token", got)
		}
		json.NewEncoder(w).Encode([]accessibleResource{
			{ID: "cloud-id-abc", Name: "My Site", URL: "https://mysite.atlassian.net"},
		})
	}))
	defer srv.Close()

	conn := newOAuthForTest(srv.Client(), srv.URL+"/accessible-resources")
	base, err := conn.oauthAPIBase(t.Context(), validOAuthCreds())
	if err != nil {
		t.Fatalf("oauthAPIBase() unexpected error: %v", err)
	}
	want := "https://api.atlassian.com/ex/jira/cloud-id-abc/rest/api/3"
	if base != want {
		t.Errorf("oauthAPIBase() = %q, want %q", base, want)
	}
}

func TestJiraConnector_OAuthAPIBase_MultipleSites(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]accessibleResource{
			{ID: "first-cloud-id", Name: "Primary Site", URL: "https://primary.atlassian.net"},
			{ID: "second-cloud-id", Name: "Secondary Site", URL: "https://secondary.atlassian.net"},
		})
	}))
	defer srv.Close()

	// Should use the first accessible resource.
	conn := newOAuthForTest(srv.Client(), srv.URL+"/accessible-resources")
	base, err := conn.oauthAPIBase(t.Context(), validOAuthCreds())
	if err != nil {
		t.Fatalf("oauthAPIBase() unexpected error: %v", err)
	}
	want := "https://api.atlassian.com/ex/jira/first-cloud-id/rest/api/3"
	if base != want {
		t.Errorf("oauthAPIBase() = %q, want %q", base, want)
	}
}

func TestJiraConnector_OAuthAPIBase_NoResources(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]accessibleResource{})
	}))
	defer srv.Close()

	conn := newOAuthForTest(srv.Client(), srv.URL+"/accessible-resources")
	creds := connectors.NewCredentials(map[string]string{"access_token": "tok"})
	_, err := conn.oauthAPIBase(t.Context(), creds)
	if err == nil {
		t.Fatal("oauthAPIBase() expected error for empty resources, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestJiraConnector_OAuthAPIBase_EmptyCloudID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]accessibleResource{
			{ID: "", Name: "Bad Site", URL: "https://bad.atlassian.net"},
		})
	}))
	defer srv.Close()

	conn := newOAuthForTest(srv.Client(), srv.URL+"/accessible-resources")
	creds := connectors.NewCredentials(map[string]string{"access_token": "tok"})
	_, err := conn.oauthAPIBase(t.Context(), creds)
	if err == nil {
		t.Fatal("oauthAPIBase() expected error for empty cloud ID, got nil")
	}
}

func TestJiraConnector_OAuthAPIBase_Unauthorized(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "invalid_token"}`))
	}))
	defer srv.Close()

	conn := newOAuthForTest(srv.Client(), srv.URL+"/accessible-resources")
	creds := connectors.NewCredentials(map[string]string{"access_token": "expired-token"})
	_, err := conn.oauthAPIBase(t.Context(), creds)
	if err == nil {
		t.Fatal("oauthAPIBase() expected error for 401, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError for 401, got %T: %v", err, err)
	}
}

func TestJiraConnector_OAuthAPIBase_Forbidden(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte(`{"error": "insufficient_scope"}`))
	}))
	defer srv.Close()

	conn := newOAuthForTest(srv.Client(), srv.URL+"/accessible-resources")
	creds := connectors.NewCredentials(map[string]string{"access_token": "bad-scope-token"})
	_, err := conn.oauthAPIBase(t.Context(), creds)
	if err == nil {
		t.Fatal("oauthAPIBase() expected error for 403, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError for 403, got %T: %v", err, err)
	}
}

func TestJiraConnector_OAuthAPIBase_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal server error"))
	}))
	defer srv.Close()

	conn := newOAuthForTest(srv.Client(), srv.URL+"/accessible-resources")
	creds := connectors.NewCredentials(map[string]string{"access_token": "tok"})
	_, err := conn.oauthAPIBase(t.Context(), creds)
	if err == nil {
		t.Fatal("oauthAPIBase() expected error for 500, got nil")
	}
	// 500 should be ExternalError, not AuthError.
	if connectors.IsAuthError(err) {
		t.Errorf("expected ExternalError for 500, got AuthError: %v", err)
	}
}

func TestJiraConnector_OAuthAPIBase_MalformedJSON(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not valid json"))
	}))
	defer srv.Close()

	conn := newOAuthForTest(srv.Client(), srv.URL+"/accessible-resources")
	creds := connectors.NewCredentials(map[string]string{"access_token": "tok"})
	_, err := conn.oauthAPIBase(t.Context(), creds)
	if err == nil {
		t.Fatal("oauthAPIBase() expected error for malformed JSON, got nil")
	}
}

func TestJiraConnector_OAuthAPIBase_CachesCloudID(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		json.NewEncoder(w).Encode([]accessibleResource{
			{ID: "cached-cloud-id", Name: "My Site", URL: "https://mysite.atlassian.net"},
		})
	}))
	defer srv.Close()

	conn := newOAuthForTest(srv.Client(), srv.URL+"/accessible-resources")
	creds := validOAuthCreds()

	// First call should hit the server.
	base1, err := conn.oauthAPIBase(t.Context(), creds)
	if err != nil {
		t.Fatalf("first oauthAPIBase() unexpected error: %v", err)
	}

	// Second call should use the cache.
	base2, err := conn.oauthAPIBase(t.Context(), creds)
	if err != nil {
		t.Fatalf("second oauthAPIBase() unexpected error: %v", err)
	}

	if base1 != base2 {
		t.Errorf("cached result differs: %q vs %q", base1, base2)
	}
	if callCount != 1 {
		t.Errorf("expected 1 server call (cached), got %d", callCount)
	}
}

func TestJiraConnector_IsOAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		creds connectors.Credentials
		want  bool
	}{
		{"with access_token", validOAuthCreds(), true},
		{"with basic auth", validCreds(), false},
		{"empty access_token", connectors.NewCredentials(map[string]string{"access_token": ""}), false},
		{"no credentials", connectors.NewCredentials(nil), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isOAuth(tt.creds); got != tt.want {
				t.Errorf("isOAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestJiraConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*JiraConnector)(nil)
	var _ connectors.ManifestProvider = (*JiraConnector)(nil)
}

func TestJiraConnector_SiteValidation_SSRF(t *testing.T) {
	t.Parallel()

	conn := New()
	maliciousSites := []struct {
		name string
		site string
	}{
		{"path traversal", "evil.com/steal-creds#"},
		{"slash in site", "my-site/../../admin"},
		{"port injection", "evil.com:8080"},
		{"dot notation", "evil.com."},
		{"fragment", "site#fragment"},
		{"query param", "site?foo=bar"},
		{"at sign", "user@evil.com"},
		{"space", "my site"},
	}

	for _, tt := range maliciousSites {
		t.Run(tt.name, func(t *testing.T) {
			creds := connectors.NewCredentials(map[string]string{
				"site":      tt.site,
				"email":     "user@example.com",
				"api_token": "token",
			})
			err := conn.do(t.Context(), creds, http.MethodGet, "/test", nil, nil)
			if err == nil {
				t.Fatalf("expected error for malicious site %q, got nil", tt.site)
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError for site %q, got %T: %v", tt.site, err, err)
			}
		})
	}
}

func TestJiraConnector_SiteValidation_ValidSites(t *testing.T) {
	t.Parallel()

	validSites := []string{"mycompany", "my-company", "company123", "a"}

	for _, site := range validSites {
		conn := New()
		creds := connectors.NewCredentials(map[string]string{
			"site":      site,
			"email":     "user@example.com",
			"api_token": "token",
		})
		_, err := conn.apiBase(t.Context(), creds)
		if err != nil {
			t.Errorf("unexpected error for valid site %q: %v", site, err)
		}
	}
}
