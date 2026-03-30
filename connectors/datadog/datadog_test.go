package datadog

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestDatadogConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "datadog" {
		t.Errorf("ID() = %q, want %q", got, "datadog")
	}
}

func TestDatadogConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"datadog.get_metrics",
		"datadog.get_incident",
		"datadog.create_incident",
		"datadog.snooze_alert",
		"datadog.trigger_runbook",
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

func TestDatadogConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		// Custom auth (api_key + app_key)
		{
			name:    "valid custom credentials",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "valid custom with site",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "k", "app_key": "a", "site": "datadoghq.eu"}),
			wantErr: false,
		},
		{
			name:    "invalid site with custom auth",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "k", "app_key": "a", "site": "invalid.example.com"}),
			wantErr: true,
		},
		{
			name:    "missing api_key",
			creds:   connectors.NewCredentials(map[string]string{"app_key": "test"}),
			wantErr: true,
		},
		{
			name:    "missing app_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "test"}),
			wantErr: true,
		},
		{
			name:    "empty api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "", "app_key": "test"}),
			wantErr: true,
		},
		{
			name:    "empty app_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "test", "app_key": ""}),
			wantErr: true,
		},
		{
			name:    "zero-value credentials",
			creds:   connectors.Credentials{},
			wantErr: true,
		},
		// OAuth auth (access_token)
		{
			name:    "valid OAuth access_token",
			creds:   validOAuthCreds(),
			wantErr: false,
		},
		{
			name:    "OAuth with valid site",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok", "site": "us3.datadoghq.com"}),
			wantErr: false,
		},
		{
			name:    "OAuth with invalid site",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok", "site": "invalid.example.com"}),
			wantErr: true,
		},
		{
			name:    "OAuth preferred over api_key+app_key when both present",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "tok", "api_key": "k", "app_key": "a"}),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("ValidateCredentials() returned %T, want *connectors.ValidationError", err)
			}
		})
	}
}

func TestDatadogConnector_SiteRouting(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"status": "ok"})
	}))
	defer srv.Close()

	// newForTest overrides the base URL, but we can test the baseURLForCreds logic directly.
	c := New()

	tests := []struct {
		name    string
		site    string
		wantURL string
	}{
		{name: "default (no site)", site: "", wantURL: "https://api.datadoghq.com"},
		{name: "EU site", site: "datadoghq.eu", wantURL: "https://api.datadoghq.eu"},
		{name: "US3 site", site: "us3.datadoghq.com", wantURL: "https://api.us3.datadoghq.com"},
		{name: "Gov site", site: "ddog-gov.com", wantURL: "https://api.ddog-gov.com"},
		{name: "unknown site falls back to default", site: "unknown.example.com", wantURL: "https://api.datadoghq.com"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := connectors.NewCredentials(map[string]string{
				"api_key": "k", "app_key": "a", "site": tt.site,
			})
			got := c.baseURLForCreds(creds)
			if got != tt.wantURL {
				t.Errorf("baseURLForCreds() = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestDatadogConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "datadog" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "datadog")
	}
	if m.Name != "Datadog" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Datadog")
	}
	if len(m.Actions) != 5 {
		t.Fatalf("Manifest().Actions has %d items, want 5", len(m.Actions))
	}

	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"datadog.get_metrics",
		"datadog.get_incident",
		"datadog.create_incident",
		"datadog.snooze_alert",
		"datadog.trigger_runbook",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}

	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 2", len(m.RequiredCredentials))
	}
	// OAuth should be first (default/primary auth method).
	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "datadog_oauth" {
		t.Errorf("RequiredCredentials[0].Service = %q, want %q", oauthCred.Service, "datadog_oauth")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("RequiredCredentials[0].AuthType = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "datadog" {
		t.Errorf("RequiredCredentials[0].OAuthProvider = %q, want %q", oauthCred.OAuthProvider, "datadog")
	}
	// OAuthScopes must exactly match the package-level OAuthScopes var —
	// they share a single source of truth so they can never drift.
	if len(oauthCred.OAuthScopes) != len(OAuthScopes) {
		t.Errorf("RequiredCredentials[0].OAuthScopes has %d entries, want %d (OAuthScopes var)", len(oauthCred.OAuthScopes), len(OAuthScopes))
	} else {
		for i, s := range OAuthScopes {
			if oauthCred.OAuthScopes[i] != s {
				t.Errorf("RequiredCredentials[0].OAuthScopes[%d] = %q, want %q", i, oauthCred.OAuthScopes[i], s)
			}
		}
	}
	// Custom auth should be second (alternative auth method).
	customCred := m.RequiredCredentials[1]
	if customCred.Service != "datadog" {
		t.Errorf("RequiredCredentials[1].Service = %q, want %q", customCred.Service, "datadog")
	}
	if customCred.AuthType != "custom" {
		t.Errorf("RequiredCredentials[1].AuthType = %q, want %q", customCred.AuthType, "custom")
	}
	if customCred.InstructionsURL == "" {
		t.Error("RequiredCredentials[1].InstructionsURL is empty, want a URL")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestDatadogConnector_ActionsMatchManifest(t *testing.T) {
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

func TestDatadogConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*DatadogConnector)(nil)
	var _ connectors.ManifestProvider = (*DatadogConnector)(nil)
}

func TestDatadogConnector_Do_UsesOAuthBearerToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// OAuth path must use Authorization: Bearer, not DD-API-KEY.
		if got := r.Header.Get("Authorization"); got != "Bearer dd_oauth_test_token_abc" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer dd_oauth_test_token_abc")
		}
		if got := r.Header.Get("DD-API-KEY"); got != "" {
			t.Errorf("DD-API-KEY should not be set for OAuth, got %q", got)
		}
		if got := r.Header.Get("DD-APPLICATION-KEY"); got != "" {
			t.Errorf("DD-APPLICATION-KEY should not be set for OAuth, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	if err := c.do(t.Context(), validOAuthCreds(), http.MethodGet, "/api/v2/metrics", nil, &resp); err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
}

func TestDatadogConnector_Do_UsesAPIKeysWhenNoAccessToken(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("DD-API-KEY"); got != "dd_test_api_key_123" {
			t.Errorf("DD-API-KEY = %q, want %q", got, "dd_test_api_key_123")
		}
		if got := r.Header.Get("DD-APPLICATION-KEY"); got != "dd_test_app_key_456" {
			t.Errorf("DD-APPLICATION-KEY = %q, want %q", got, "dd_test_app_key_456")
		}
		if got := r.Header.Get("Authorization"); got != "" {
			t.Errorf("Authorization should not be set for custom auth, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	if err := c.do(t.Context(), validCreds(), http.MethodGet, "/api/v2/metrics", nil, &resp); err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
}

func TestDatadogConnector_Do_PrefersOAuthOverAPIKeys(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// When both access_token and api_key are present, Bearer wins.
		if got := r.Header.Get("Authorization"); got != "Bearer oauth_wins" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer oauth_wins")
		}
		if got := r.Header.Get("DD-API-KEY"); got != "" {
			t.Errorf("DD-API-KEY should not be set when access_token present, got %q", got)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	creds := connectors.NewCredentials(map[string]string{
		"access_token": "oauth_wins",
		"api_key":      "should_be_ignored",
		"app_key":      "should_be_ignored",
	})
	c := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	if err := c.do(t.Context(), creds, http.MethodGet, "/api/v2/metrics", nil, &resp); err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
}
