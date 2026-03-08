package zendesk

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestZendeskConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "zendesk" {
		t.Errorf("ID() = %q, want %q", got, "zendesk")
	}
}

func TestZendeskConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"zendesk.create_ticket",
		"zendesk.reply_ticket",
		"zendesk.update_ticket",
		"zendesk.assign_ticket",
		"zendesk.merge_tickets",
		"zendesk.search_tickets",
		"zendesk.list_tags",
		"zendesk.update_tags",
		"zendesk.create_user",
		"zendesk.get_user",
		"zendesk.list_ticket_fields",
		"zendesk.get_satisfaction_ratings",
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

func TestZendeskConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid API token credentials",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "valid OAuth credentials",
			creds:   connectors.NewCredentials(map[string]string{"subdomain": "test", "access_token": "oauth-token"}),
			wantErr: false,
		},
		{
			name:    "missing subdomain with OAuth",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "oauth-token"}),
			wantErr: true,
		},
		{
			name:    "missing subdomain",
			creds:   connectors.NewCredentials(map[string]string{"email": "a@b.com", "api_token": "tok"}),
			wantErr: true,
		},
		{
			name:    "missing email",
			creds:   connectors.NewCredentials(map[string]string{"subdomain": "test", "api_token": "tok"}),
			wantErr: true,
		},
		{
			name:    "missing api_token",
			creds:   connectors.NewCredentials(map[string]string{"subdomain": "test", "email": "a@b.com"}),
			wantErr: true,
		},
		{
			name:    "invalid subdomain format",
			creds:   connectors.NewCredentials(map[string]string{"subdomain": "test company!", "email": "a@b.com", "api_token": "tok"}),
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

func TestZendeskConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "zendesk" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "zendesk")
	}
	if m.Name != "Zendesk" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Zendesk")
	}
	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 2", len(m.RequiredCredentials))
	}
	// First credential should be OAuth (primary/default method).
	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "zendesk" {
		t.Errorf("oauth credential service = %q, want %q", oauthCred.Service, "zendesk")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("oauth credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "zendesk" {
		t.Errorf("oauth credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "zendesk")
	}
	// Second credential should be API token (legacy fallback).
	apiCred := m.RequiredCredentials[1]
	if apiCred.Service != "zendesk" {
		t.Errorf("api credential service = %q, want %q", apiCred.Service, "zendesk")
	}
	if apiCred.AuthType != "custom" {
		t.Errorf("api credential auth_type = %q, want %q", apiCred.AuthType, "custom")
	}
	if apiCred.InstructionsURL == "" {
		t.Error("api credential instructions_url is empty, want a URL")
	}

	if len(m.Actions) != 12 {
		t.Fatalf("Manifest().Actions has %d items, want 12", len(m.Actions))
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestZendeskConnector_ActionsMatchManifest(t *testing.T) {
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

func TestZendeskConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*ZendeskConnector)(nil)
	var _ connectors.ManifestProvider = (*ZendeskConnector)(nil)
}

func TestZendeskConnector_Do_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Error("expected basic auth")
		}
		if user != "agent@example.com/token" {
			t.Errorf("expected user 'agent@example.com/token', got %q", user)
		}
		if pass != "test-api-token-123" {
			t.Errorf("expected pass 'test-api-token-123', got %q", pass)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("expected Accept application/json, got %q", got)
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

func TestZendeskConnector_Do_OAuth(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer oauth-access-token" {
			t.Errorf("expected Bearer auth, got %q", authHeader)
		}
		// Should not have basic auth when using OAuth.
		if _, _, ok := r.BasicAuth(); ok {
			t.Error("did not expect basic auth with OAuth credentials")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"ok": true})
	}))
	defer srv.Close()

	oauthCreds := connectors.NewCredentials(map[string]string{
		"subdomain":    "testcompany",
		"access_token": "oauth-access-token",
	})
	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.do(context.Background(), oauthCreds, http.MethodGet, "/test", nil, &resp)
	if err != nil {
		t.Fatalf("do() with OAuth unexpected error: %v", err)
	}
	if resp["ok"] != true {
		t.Errorf("response ok = %v, want true", resp["ok"])
	}
}

func TestZendeskConnector_Do_NilBodies(t *testing.T) {
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

func TestZendeskConnector_Do_Timeout(t *testing.T) {
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
