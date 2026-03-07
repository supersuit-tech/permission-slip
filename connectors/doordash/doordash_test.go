package doordash

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDoorDashConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "doordash" {
		t.Errorf("ID() = %q, want %q", got, "doordash")
	}
}

func TestDoorDashConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	wantActions := []string{
		"doordash.get_quote",
		"doordash.create_delivery",
		"doordash.get_delivery",
		"doordash.cancel_delivery",
		"doordash.list_deliveries",
	}
	if len(actions) != len(wantActions) {
		t.Fatalf("Actions() returned %d actions, want %d", len(actions), len(wantActions))
	}
	for _, name := range wantActions {
		if _, ok := actions[name]; !ok {
			t.Errorf("Actions() missing %q", name)
		}
	}
}

func TestDoorDashConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid credentials",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "missing developer_id",
			creds:   connectors.NewCredentials(map[string]string{"key_id": "k", "signing_secret": "s"}),
			wantErr: true,
		},
		{
			name:    "missing key_id",
			creds:   connectors.NewCredentials(map[string]string{"developer_id": "d", "signing_secret": "s"}),
			wantErr: true,
		},
		{
			name:    "missing signing_secret",
			creds:   connectors.NewCredentials(map[string]string{"developer_id": "d", "key_id": "k"}),
			wantErr: true,
		},
		{
			name:    "empty credentials",
			creds:   connectors.NewCredentials(map[string]string{}),
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

func TestDoorDashConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "doordash" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "doordash")
	}
	if m.Name != "DoorDash Drive" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "DoorDash Drive")
	}
	if len(m.Actions) != 5 {
		t.Fatalf("Manifest().Actions has %d items, want 5", len(m.Actions))
	}
	wantActions := []string{
		"doordash.get_quote",
		"doordash.create_delivery",
		"doordash.get_delivery",
		"doordash.cancel_delivery",
		"doordash.list_deliveries",
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range wantActions {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "doordash" {
		t.Errorf("credential service = %q, want %q", cred.Service, "doordash")
	}
	if cred.AuthType != "api_key" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "api_key")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}

	for _, a := range m.Actions {
		if len(a.ParametersSchema) == 0 {
			continue
		}
		var schema map[string]interface{}
		if err := json.Unmarshal(a.ParametersSchema, &schema); err != nil {
			t.Errorf("action %q has invalid ParametersSchema JSON: %v", a.ActionType, err)
		}
	}
}

func TestDoorDashConnector_ActionsMatchManifest(t *testing.T) {
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

func TestDoorDashConnector_ManifestTemplates(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if len(m.Templates) == 0 {
		t.Fatal("Manifest().Templates is empty, want at least one template")
	}

	actionTypes := make(map[string]bool, len(m.Actions))
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}

	seenIDs := make(map[string]bool)
	for _, tpl := range m.Templates {
		if seenIDs[tpl.ID] {
			t.Errorf("duplicate template ID %q", tpl.ID)
		}
		seenIDs[tpl.ID] = true

		if !actionTypes[tpl.ActionType] {
			t.Errorf("template %q references unknown action_type %q", tpl.ID, tpl.ActionType)
		}
		if tpl.Name == "" {
			t.Errorf("template %q has empty name", tpl.ID)
		}
		if len(tpl.Parameters) == 0 {
			t.Errorf("template %q has empty parameters", tpl.ID)
		}
		var params map[string]interface{}
		if err := json.Unmarshal(tpl.Parameters, &params); err != nil {
			t.Errorf("template %q has invalid Parameters JSON: %v", tpl.ID, err)
		}
	}
}

func TestDoorDashConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*DoorDashConnector)(nil)
	var _ connectors.ManifestProvider = (*DoorDashConnector)(nil)
}

func TestDoorDashConnector_Do_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/test/path" {
			t.Errorf("path = %s, want /test/path", r.URL.Path)
		}
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			t.Errorf("Authorization = %q, want Bearer prefix", auth)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["key"] != "value" {
			t.Errorf("request body key = %q, want %q", reqBody["key"], "value")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "abc123"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodPost, "/test/path", map[string]string{"key": "value"}, &resp)

	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["id"] != "abc123" {
		t.Errorf("response id = %q, want %q", resp["id"], "abc123")
	}
}

func TestDoorDashConnector_Do_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"message": "Invalid JWT token"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestDoorDashConnector_Do_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{"message": "Too many requests"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestDoorDashConnector_Do_ValidationError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"message":      "Invalid request",
			"field_errors": []map[string]string{{"field": "pickup_address", "error": "is required"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodPost, "/test", map[string]string{}, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestDoorDashConnector_Do_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
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

func TestDoorDashConnector_Do_MissingCredentials(t *testing.T) {
	t.Parallel()

	conn := New()
	err := conn.do(t.Context(), connectors.Credentials{}, http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGenerateJWT(t *testing.T) {
	t.Parallel()

	token, err := generateJWT(validCreds())
	if err != nil {
		t.Fatalf("generateJWT() unexpected error: %v", err)
	}
	if token == "" {
		t.Error("generateJWT() returned empty token")
	}

	// Verify the token has 3 parts (header.payload.signature).
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Errorf("JWT has %d parts, want 3", len(parts))
	}
}

func TestGenerateJWT_MissingCredentials(t *testing.T) {
	t.Parallel()

	_, err := generateJWT(connectors.Credentials{})
	if err == nil {
		t.Fatal("generateJWT() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGenerateJWT_InvalidBase64Secret(t *testing.T) {
	t.Parallel()

	creds := connectors.NewCredentials(map[string]string{
		"developer_id":   "dev-123",
		"key_id":         "key-456",
		"signing_secret": "not-valid-base64-!!!",
	})
	_, err := generateJWT(creds)
	if err == nil {
		t.Fatal("generateJWT() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestNewUUID(t *testing.T) {
	t.Parallel()

	id1 := newUUID()
	id2 := newUUID()

	if id1 == "" {
		t.Error("newUUID() returned empty string")
	}
	if id1 == id2 {
		t.Errorf("newUUID() returned duplicate IDs: %q", id1)
	}
	if len(id1) != 36 {
		t.Errorf("newUUID() length = %d, want 36", len(id1))
	}
	if id1[14] != '4' {
		t.Errorf("newUUID() version nibble = %c, want '4'", id1[14])
	}
}
