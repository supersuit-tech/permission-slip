package hubspot

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestHubSpotConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "hubspot" {
		t.Errorf("ID() = %q, want %q", got, "hubspot")
	}
}

func TestHubSpotConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	// Phase 1: no actions registered yet.
	if len(actions) != 0 {
		t.Errorf("Actions() returned %d actions, want 0 (Phase 1 scaffold)", len(actions))
	}
}

func TestHubSpotConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "pat-na1-test-token-123"}),
			wantErr: false,
		},
		{
			name:    "missing api_key",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": ""}),
			wantErr: true,
		},
		{
			name:    "wrong key name",
			creds:   connectors.NewCredentials(map[string]string{"token": "pat-na1-test-token-123"}),
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

func TestHubSpotConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "hubspot" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "hubspot")
	}
	if m.Name != "HubSpot" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "HubSpot")
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "hubspot" {
		t.Errorf("credential service = %q, want %q", cred.Service, "hubspot")
	}
	if cred.AuthType != "api_key" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "api_key")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	// Full Validate() requires actions, which are added in Phase 2.
	// For now, verify the fields we have are well-formed.
}

func TestHubSpotConnector_ActionsMatchManifest(t *testing.T) {
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

func TestHubSpotConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*HubSpotConnector)(nil)
	var _ connectors.ManifestProvider = (*HubSpotConnector)(nil)
}

func TestHubSpotConnector_Do_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/contacts" {
			t.Errorf("expected path /crm/v3/objects/contacts, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer pat-na1-test-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected Content-Type application/json, got %q", got)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("expected Accept application/json, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "123",
			"properties": map[string]string{
				"email": "test@example.com",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.do(context.Background(), validCreds(), http.MethodPost, "/crm/v3/objects/contacts",
		map[string]any{"properties": map[string]string{"email": "test@example.com"}}, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["id"] != "123" {
		t.Errorf("response id = %v, want %q", resp["id"], "123")
	}
}

func TestHubSpotConnector_Do_NilBodies(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
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

func TestHubSpotConnector_Do_MissingCredentials(t *testing.T) {
	t.Parallel()

	conn := New()
	err := conn.do(context.Background(), connectors.Credentials{}, http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("do() with empty credentials expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("do() with empty credentials returned %T, want *connectors.ValidationError", err)
	}
}

func TestHubSpotConnector_Do_Timeout(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately to simulate timeout

	conn := New()
	err := conn.do(ctx, validCreds(), http.MethodGet, "http://localhost:1/test", nil, nil)
	if err == nil {
		t.Fatal("do() with cancelled context expected error, got nil")
	}
}
