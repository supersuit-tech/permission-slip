package confluence

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
	_ connectors.Connector        = (*ConfluenceConnector)(nil)
	_ connectors.ManifestProvider = (*ConfluenceConnector)(nil)
)

func TestConfluenceConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "confluence" {
		t.Errorf("ID() = %q, want %q", c.ID(), "confluence")
	}
}

func TestConfluenceConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid credentials",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "missing site",
			creds:   connectors.NewCredentials(map[string]string{"email": "user@example.com", "api_token": "tok"}),
			wantErr: true,
			errMsg:  "site",
		},
		{
			name:    "missing email",
			creds:   connectors.NewCredentials(map[string]string{"site": "mysite", "api_token": "tok"}),
			wantErr: true,
			errMsg:  "email",
		},
		{
			name:    "missing api_token",
			creds:   connectors.NewCredentials(map[string]string{"site": "mysite", "email": "user@example.com"}),
			wantErr: true,
			errMsg:  "api_token",
		},
		{
			name:    "empty site",
			creds:   connectors.NewCredentials(map[string]string{"site": "", "email": "user@example.com", "api_token": "tok"}),
			wantErr: true,
			errMsg:  "site",
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

func TestConfluenceConnector_Do_BasicAuth(t *testing.T) {
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

func TestConfluenceConnector_Do_PostBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "12345"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	body := map[string]string{"title": "Test page"}
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodPost, "/pages", body, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["id"] != "12345" {
		t.Errorf("id = %q, want %q", resp["id"], "12345")
	}
}

func TestConfluenceConnector_Do_Timeout(t *testing.T) {
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

func TestConfluenceConnector_Do_ContextCanceled(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // cancel immediately

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(ctx, validCreds(), http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError for canceled context, got %T: %v", err, err)
	}
}

func TestConfluenceConnector_Do_MissingCredentials(t *testing.T) {
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

func TestConfluenceConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"confluence.create_page",
		"confluence.update_page",
		"confluence.get_page",
		"confluence.search",
		"confluence.add_comment",
		"confluence.list_spaces",
		"confluence.list_pages",
		"confluence.delete_page",
		"confluence.get_attachments",
		"confluence.add_attachment",
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

func TestConfluenceConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "confluence" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "confluence")
	}
	if m.Name != "Confluence" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Confluence")
	}
	if len(m.Actions) != 10 {
		t.Fatalf("Manifest().Actions has %d items, want 10", len(m.Actions))
	}

	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"confluence.create_page", "confluence.update_page", "confluence.get_page",
		"confluence.search", "confluence.add_comment",
		"confluence.list_spaces", "confluence.list_pages", "confluence.delete_page",
		"confluence.get_attachments", "confluence.add_attachment",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}

	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "jira" {
		t.Errorf("credential service = %q, want %q", cred.Service, "jira")
	}
	if cred.AuthType != "basic" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "basic")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestConfluenceConnector_ActionsMatchManifest(t *testing.T) {
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

func TestConfluenceConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*ConfluenceConnector)(nil)
	var _ connectors.ManifestProvider = (*ConfluenceConnector)(nil)
}

func TestConfluenceConnector_SiteValidation_SSRF(t *testing.T) {
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

func TestConfluenceConnector_SiteValidation_ValidSites(t *testing.T) {
	t.Parallel()

	validSites := []string{"mycompany", "my-company", "company123", "a"}

	for _, site := range validSites {
		conn := New()
		creds := connectors.NewCredentials(map[string]string{
			"site":      site,
			"email":     "user@example.com",
			"api_token": "token",
		})
		_, err := conn.apiBase(creds)
		if err != nil {
			t.Errorf("unexpected error for valid site %q: %v", site, err)
		}
	}
}
