package figma

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestFigmaConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "figma" {
		t.Errorf("expected ID 'figma', got %q", c.ID())
	}
}

func TestFigmaConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"figma.get_file",
		"figma.get_components",
		"figma.export_images",
		"figma.list_comments",
		"figma.post_comment",
		"figma.get_versions",
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

func TestFigmaConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid token",
			creds:   connectors.NewCredentials(map[string]string{"personal_access_token": "figd_1234567890abcdef"}),
			wantErr: false,
		},
		{
			name:    "missing token",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty token",
			creds:   connectors.NewCredentials(map[string]string{"personal_access_token": ""}),
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

func TestFigmaConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "figma" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "figma")
	}
	if m.Name != "Figma" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Figma")
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "figma" {
		t.Errorf("credential service = %q, want %q", cred.Service, "figma")
	}
	if cred.AuthType != "custom" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "custom")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	if len(m.Actions) != 6 {
		t.Fatalf("Manifest().Actions has %d items, want 6", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"figma.get_file", "figma.get_components", "figma.export_images",
		"figma.list_comments", "figma.post_comment", "figma.get_versions",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestFigmaConnector_ActionsMatchManifest(t *testing.T) {
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

func TestFigmaConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*FigmaConnector)(nil)
	var _ connectors.ManifestProvider = (*FigmaConnector)(nil)
}

func TestFigmaConnector_RedirectStripsToken(t *testing.T) {
	t.Parallel()

	// tokenLeaked is set atomically by the evil server's handler goroutine
	// and read by the test goroutine to avoid a data race.
	var tokenLeaked atomic.Bool
	evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Figma-Token") != "" {
			tokenLeaked.Store(true)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"ok":true}`)
	}))
	t.Cleanup(evil.Close)

	// "figma" server that redirects to the evil server
	figmaSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, evil.URL+"/stolen", http.StatusFound)
	}))
	t.Cleanup(figmaSrv.Close)

	conn := &FigmaConnector{
		client: &http.Client{
			CheckRedirect: safeRedirectPolicy(figmaSrv.URL),
		},
		baseURL: figmaSrv.URL,
	}

	var dest map[string]any
	_ = conn.doGet(t.Context(), "/files/abc", validCreds(), &dest)

	if tokenLeaked.Load() {
		t.Error("X-Figma-Token was leaked to cross-origin redirect target")
	}
}

func TestFigmaConnector_DoGet_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("X-Figma-Token"); got != "figd_test_token_123" {
			t.Errorf("bad auth header: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"name":"Test File","document":{"id":"0:0"}}`)
	})

	var dest map[string]any
	err := conn.doGet(t.Context(), "/files/abc123", validCreds(), &dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest["name"] != "Test File" {
		t.Errorf("expected name 'Test File', got %v", dest["name"])
	}
}

func TestFigmaConnector_DoGet_AuthError(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		w.Write(figmaErrorBody(403, "Forbidden"))
	})

	var dest map[string]any
	err := conn.doGet(t.Context(), "/files/abc123", validCreds(), &dest)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestFigmaConnector_DoGet_RateLimited(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
	})

	var dest map[string]any
	err := conn.doGet(t.Context(), "/files/abc123", validCreds(), &dest)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestFigmaConnector_DoGet_MissingCreds(t *testing.T) {
	t.Parallel()
	conn := New()

	var dest map[string]any
	err := conn.doGet(t.Context(), "/files/abc123", connectors.Credentials{}, &dest)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestFigmaConnector_DoGet_NotFound(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write(figmaErrorBody(404, "Not found"))
	})

	var dest map[string]any
	err := conn.doGet(t.Context(), "/files/abc123", validCreds(), &dest)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}

func TestFigmaConnector_DoGet_ServerError(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		w.Write(figmaErrorBody(500, "Internal server error"))
	})

	var dest map[string]any
	err := conn.doGet(t.Context(), "/files/abc123", validCreds(), &dest)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestFigmaConnector_DoPost_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("X-Figma-Token"); got != "figd_test_token_123" {
			t.Errorf("bad auth header: %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("bad Content-Type: %q", got)
		}

		var body map[string]string
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"id":"comment-1","message":%q}`, body["message"])
	})

	var dest map[string]any
	err := conn.doPost(t.Context(), "/files/abc123/comments", validCreds(), map[string]string{"message": "test"}, &dest)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if dest["id"] != "comment-1" {
		t.Errorf("expected id 'comment-1', got %v", dest["id"])
	}
}

func TestMapFigmaHTTPError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status  int
		checker func(error) bool
	}{
		{401, connectors.IsAuthError},
		{403, connectors.IsAuthError},
		{404, connectors.IsValidationError},
		{400, connectors.IsValidationError},
		{429, connectors.IsRateLimitError},
		{500, connectors.IsExternalError},
		{503, connectors.IsExternalError},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("status_%d", tt.status), func(t *testing.T) {
			body := figmaErrorBody(tt.status, "test error")
			err := mapFigmaHTTPError(tt.status, body)
			if !tt.checker(err) {
				t.Errorf("mapFigmaHTTPError(%d) returned %T, unexpected type", tt.status, err)
			}
		})
	}
}

func TestMapFigmaHTTPError_InvalidJSON(t *testing.T) {
	t.Parallel()

	err := mapFigmaHTTPError(500, []byte("not json"))
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError for invalid JSON, got %T: %v", err, err)
	}
}

func TestExtractFileKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		raw  string
		want string
	}{
		{"raw key", "abc123DEF", "abc123DEF"},
		{"design URL", "https://www.figma.com/design/abc123DEF/My-Design", "abc123DEF"},
		{"file URL", "https://www.figma.com/file/xyz789/Untitled", "xyz789"},
		{"no www", "https://figma.com/design/abc123DEF/My-Design", "abc123DEF"},
		{"http URL", "http://www.figma.com/design/abc123DEF/Test", "abc123DEF"},
		{"URL with query", "https://www.figma.com/design/abc123DEF/Test?node-id=1:2", "abc123DEF"},
		{"not a figma URL", "https://example.com/design/abc123", "https://example.com/design/abc123"},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFileKey(tt.raw)
			if got != tt.want {
				t.Errorf("extractFileKey(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestValidateFileKey(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{"valid key", "abc123DEF", false},
		{"empty", "", true},
		{"slash", "abc/def", true},
		{"dot-dot", "abc/../def", true},
		{"backslash", "abc\\def", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFileKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateFileKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T", err)
			}
		})
	}
}

func TestValidateNodeIDs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ids     string
		wantErr bool
	}{
		{"single", "1:2", false},
		{"multiple", "1:2,3:4", false},
		{"with spaces", "1:2, 3:4", false},
		{"empty", "", true},
		{"empty element", "1:2,,3:4", true},
		{"invalid format no colon", "abc", true},
		{"invalid format letters", "a:b", true},
		{"missing part", "1:", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNodeIDs(tt.ids)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateNodeIDs(%q) error = %v, wantErr %v", tt.ids, err, tt.wantErr)
			}
		})
	}
}
