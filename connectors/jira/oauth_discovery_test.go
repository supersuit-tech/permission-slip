package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestOAuthAPIBase_DiscoverCloudID(t *testing.T) {
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

func TestOAuthAPIBase_MultipleSites(t *testing.T) {
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

func TestOAuthAPIBase_NoResources(t *testing.T) {
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

func TestOAuthAPIBase_EmptyCloudID(t *testing.T) {
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

func TestOAuthAPIBase_Unauthorized(t *testing.T) {
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

func TestOAuthAPIBase_Forbidden(t *testing.T) {
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

func TestOAuthAPIBase_ServerError(t *testing.T) {
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

func TestOAuthAPIBase_MalformedJSON(t *testing.T) {
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

func TestOAuthAPIBase_CachesCloudID(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount.Add(1)
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
	if got := callCount.Load(); got != 1 {
		t.Errorf("expected 1 server call (cached), got %d", got)
	}
}

func TestOAuthAPIBase_CacheEviction(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]accessibleResource{
			{ID: "cloud-id", Name: "Site", URL: "https://site.atlassian.net"},
		})
	}))
	defer srv.Close()

	conn := newOAuthForTest(srv.Client(), srv.URL+"/accessible-resources")

	// Fill the cache to the limit with expired entries.
	now := time.Now()
	conn.cloudIDMu.Lock()
	for i := 0; i < maxCloudIDCacheSize; i++ {
		conn.cloudIDCache[fmt.Sprintf("fp-%d", i)] = cloudIDEntry{
			cloudID:   "old-cloud-id",
			expiresAt: now.Add(-1 * time.Hour), // expired
		}
	}
	conn.cloudIDMu.Unlock()

	// Next call should trigger eviction of expired entries and succeed.
	creds := validOAuthCreds()
	_, err := conn.oauthAPIBase(t.Context(), creds)
	if err != nil {
		t.Fatalf("oauthAPIBase() unexpected error after eviction: %v", err)
	}

	// Cache should have been cleaned: only the new entry remains.
	conn.cloudIDMu.RLock()
	size := len(conn.cloudIDCache)
	conn.cloudIDMu.RUnlock()
	if size != 1 {
		t.Errorf("cache size after eviction = %d, want 1", size)
	}
}

func TestIsOAuth(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
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
