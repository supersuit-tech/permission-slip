package notion

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestNotionConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "notion" {
		t.Errorf("expected ID 'notion', got %q", c.ID())
	}
}

func TestNotionConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"notion.create_page",
		"notion.update_page",
		"notion.append_blocks",
		"notion.query_database",
		"notion.search",
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

func TestNotionConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "ntn_1234567890abcdef"}),
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
			name:    "zero-value credentials",
			creds:   connectors.Credentials{},
			wantErr: true,
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

func TestNotionConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "notion" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "notion")
	}
	if m.Name != "Notion" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Notion")
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "notion" {
		t.Errorf("credential service = %q, want %q", cred.Service, "notion")
	}
	if cred.AuthType != "api_key" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "api_key")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	if len(m.Actions) != 5 {
		t.Fatalf("Manifest().Actions has %d items, want 5", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"notion.create_page", "notion.update_page", "notion.append_blocks",
		"notion.query_database", "notion.search",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestNotionConnector_ActionsMatchManifest(t *testing.T) {
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

func TestNotionConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*NotionConnector)(nil)
	var _ connectors.ManifestProvider = (*NotionConnector)(nil)
}

func TestNotionConnector_Do_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ntn_test_token_123" {
			t.Errorf("bad auth header: %q", got)
		}
		if got := r.Header.Get("Notion-Version"); got != notionVersion {
			t.Errorf("bad Notion-Version header: %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("bad Content-Type header: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"object":"page","id":"page-123"}`)
	})

	var dest map[string]string
	err := conn.do(t.Context(), http.MethodPost, "/v1/pages", validCreds(), map[string]string{"title": "test"}, &dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest["id"] != "page-123" {
		t.Errorf("expected id 'page-123', got %q", dest["id"])
	}
}

func TestNotionConnector_Do_AuthError(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		w.Write(notionErrorBody(401, "unauthorized", "API token is invalid."))
	})

	var dest map[string]any
	err := conn.do(t.Context(), http.MethodPost, "/v1/pages", validCreds(), map[string]string{}, &dest)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestNotionConnector_Do_ValidationError(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(notionErrorBody(400, "validation_error", "Title is not provided."))
	})

	var dest map[string]any
	err := conn.do(t.Context(), http.MethodPost, "/v1/pages", validCreds(), map[string]string{}, &dest)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestNotionConnector_Do_RateLimited(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	var dest map[string]any
	err := conn.do(t.Context(), http.MethodPost, "/v1/search", validCreds(), map[string]string{}, &dest)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestNotionConnector_Do_MissingCreds(t *testing.T) {
	t.Parallel()
	conn := New()

	var dest map[string]any
	err := conn.do(t.Context(), http.MethodPost, "/v1/pages", connectors.Credentials{}, nil, &dest)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestNotionConnector_Do_RestrictedResource(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write(notionErrorBody(404, "object_not_found", "Could not find page."))
	})

	var dest map[string]any
	err := conn.do(t.Context(), http.MethodGet, "/v1/pages/abc", validCreds(), nil, &dest)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError for object_not_found, got %T: %v", err, err)
	}
}

func TestNotionConnector_Do_ServerError(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(notionErrorBody(500, "internal_server_error", "Internal server error"))
	})

	var dest map[string]any
	err := conn.do(t.Context(), http.MethodPost, "/v1/pages", validCreds(), map[string]string{}, &dest)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestMapNotionError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		code    string
		checker func(error) bool
	}{
		{"unauthorized", connectors.IsAuthError},
		{"restricted_resource", connectors.IsAuthError},
		{"object_not_found", connectors.IsAuthError},
		{"validation_error", connectors.IsValidationError},
		{"rate_limited", connectors.IsRateLimitError},
		{"internal_server_error", connectors.IsExternalError},
		{"service_unavailable", connectors.IsExternalError},
		{"conflict_error", connectors.IsExternalError},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			body, _ := json.Marshal(notionErrorResponse{
				Object:  "error",
				Status:  400,
				Code:    tt.code,
				Message: "test error",
			})
			err := mapNotionHTTPError(400, body)
			if !tt.checker(err) {
				t.Errorf("mapNotionHTTPError(%q) returned %T, unexpected type", tt.code, err)
			}
		})
	}
}

func TestMapNotionHTTPError_InvalidJSON(t *testing.T) {
	t.Parallel()

	err := mapNotionHTTPError(500, []byte("not json"))
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError for invalid JSON, got %T: %v", err, err)
	}
}
