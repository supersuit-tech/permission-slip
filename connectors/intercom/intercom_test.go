package intercom

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestIntercomConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "intercom" {
		t.Errorf("ID() = %q, want %q", got, "intercom")
	}
}

func TestIntercomConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"intercom.create_ticket",
		"intercom.reply_ticket",
		"intercom.update_ticket",
		"intercom.assign_ticket",
		"intercom.search_tickets",
		"intercom.list_tags",
		"intercom.tag_ticket",
		"intercom.create_contact",
		"intercom.update_contact",
		"intercom.search_contacts",
		"intercom.send_message",
		"intercom.list_conversations",
		"intercom.create_article",
	}
	for _, name := range expected {
		if _, ok := actions[name]; !ok {
			t.Errorf("expected action %q to be registered", name)
		}
	}
	if len(actions) != len(expected) {
		t.Errorf("expected %d actions, got %d", len(expected), len(actions))
	}
}

func TestIntercomConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid access_token",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "missing access_token",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": ""}),
			wantErr: true,
		},
		{
			name:    "wrong key name",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "test"}),
			wantErr: true,
		},
		{
			name:    "zero-value credentials",
			creds:   connectors.Credentials{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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

func TestIntercomConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "intercom" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "intercom")
	}
	if m.Name != "Intercom" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Intercom")
	}
	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 2", len(m.RequiredCredentials))
	}

	// First credential: OAuth (preferred auth method).
	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "intercom_oauth" {
		t.Errorf("oauth credential service = %q, want %q", oauthCred.Service, "intercom_oauth")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("oauth credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "intercom" {
		t.Errorf("oauth credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "intercom")
	}

	// Second credential: API key (fallback).
	apiKeyCred := m.RequiredCredentials[1]
	if apiKeyCred.Service != "intercom" {
		t.Errorf("api_key credential service = %q, want %q", apiKeyCred.Service, "intercom")
	}
	if apiKeyCred.AuthType != "api_key" {
		t.Errorf("api_key credential auth_type = %q, want %q", apiKeyCred.AuthType, "api_key")
	}
	if apiKeyCred.InstructionsURL == "" {
		t.Error("api_key credential instructions_url is empty, want a URL")
	}

	// OAuth provider should be declared.
	if len(m.OAuthProviders) != 1 {
		t.Fatalf("Manifest().OAuthProviders has %d items, want 1", len(m.OAuthProviders))
	}
	if m.OAuthProviders[0].ID != "intercom" {
		t.Errorf("oauth provider ID = %q, want %q", m.OAuthProviders[0].ID, "intercom")
	}

	if len(m.Actions) != 13 {
		t.Fatalf("Manifest().Actions has %d items, want 13", len(m.Actions))
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestIntercomConnector_ActionsMatchManifest(t *testing.T) {
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

func TestIntercomConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*IntercomConnector)(nil)
	var _ connectors.ManifestProvider = (*IntercomConnector)(nil)
}

func TestIntercomConnector_Do_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer dG9rOmFiY2RlZmcxMjM0NTY3ODk=" {
			t.Errorf("expected Bearer token, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("expected Accept application/json, got %q", got)
		}
		if got := r.Header.Get("Intercom-Version"); got != "2.11" {
			t.Errorf("expected Intercom-Version 2.11, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.do(context.Background(), validCreds(), http.MethodGet, "/test", nil, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("response ok = %v, want true", resp["ok"])
	}
}

func TestIntercomConnector_Do_NilBodies(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "" {
			t.Errorf("expected no Content-Type for nil body, got %q", got)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(context.Background(), validCreds(), http.MethodGet, "/test", nil, nil)
	if err != nil {
		t.Fatalf("do() with nil bodies unexpected error: %v", err)
	}
}

func TestIntercomConnector_Do_MissingCredentials(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("request should not have been made")
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(context.Background(), connectors.Credentials{}, http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("do() with empty credentials expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestIsValidIntercomID(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id    string
		valid bool
	}{
		{"123", true},
		{"abc-def", true},
		{"ticket_42", true},
		{"", false},
		{"foo/bar", false},
		{"../admin", false},
		{"id?query=1", false},
		{"id#fragment", false},
		{"path\\traversal", false},
	}
	for _, tt := range tests {
		if got := isValidIntercomID(tt.id); got != tt.valid {
			t.Errorf("isValidIntercomID(%q) = %v, want %v", tt.id, got, tt.valid)
		}
	}
}

func TestIntercomConnector_Do_Timeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	conn := newForTest(&http.Client{}, "http://localhost:1")
	err := conn.do(ctx, validCreds(), http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("do() with cancelled context expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("do() with cancelled context should return TimeoutError, got: %T", err)
	}
}
