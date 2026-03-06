package square

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSquareConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "square" {
		t.Errorf("ID() = %q, want %q", got, "square")
	}
}

func TestSquareConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	// Phase 1 scaffold: no action handlers yet. Handlers are added in Phase 2.
	// The manifest declares 6 actions for DB seeding, but the Actions() map
	// will be populated as each action is implemented.
	if len(actions) != 0 {
		t.Errorf("Actions() returned %d actions, want 0 (Phase 1 scaffold)", len(actions))
	}
}

func TestSquareConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "EAAAEtest123"}),
			wantErr: false,
		},
		{
			name:    "valid with sandbox environment",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "EAAAEtest123", "environment": "sandbox"}),
			wantErr: false,
		},
		{
			name:    "valid with production environment",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "EAAAEtest123", "environment": "production"}),
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
			creds:   connectors.NewCredentials(map[string]string{"api_key": "EAAAEtest123"}),
			wantErr: true,
		},
		{
			name:    "invalid environment",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "EAAAEtest123", "environment": "staging"}),
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

func TestSquareConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "square" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "square")
	}
	if m.Name != "Square" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Square")
	}
	if len(m.Actions) != 6 {
		t.Fatalf("Manifest().Actions has %d items, want 6", len(m.Actions))
	}
	wantActions := []string{
		"square.create_order",
		"square.create_payment",
		"square.list_catalog",
		"square.create_customer",
		"square.create_booking",
		"square.search_orders",
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
	if cred.Service != "square" {
		t.Errorf("credential service = %q, want %q", cred.Service, "square")
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

	// Verify each action's ParametersSchema is valid JSON. This catches
	// broken fmt.Sprintf interpolation (e.g. moneySchema) that would
	// only surface at runtime otherwise.
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

func TestSquareConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*SquareConnector)(nil)
	var _ connectors.ManifestProvider = (*SquareConnector)(nil)
}

func TestSquareConnector_Do_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/test/path" {
			t.Errorf("path = %s, want /test/path", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer EAAAEBcXzPsWbM0yJjRvxlT_test" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer EAAAEBcXzPsWbM0yJjRvxlT_test")
		}
		if got := r.Header.Get("Square-Version"); got != squareVersion {
			t.Errorf("Square-Version = %q, want %q", got, squareVersion)
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

func TestSquareConnector_Do_NoContentType_ForGET(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Content-Type"); got != "" {
			t.Errorf("Content-Type = %q, want empty for GET", got)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"ok": "true"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", nil, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
}

func TestSquareConnector_Do_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "AUTHENTICATION_ERROR", "code": "UNAUTHORIZED", "detail": "This request could not be authorized."},
			},
		})
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

func TestSquareConnector_Do_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "RATE_LIMIT_ERROR", "code": "RATE_LIMITED", "detail": "Too many requests."},
			},
		})
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
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 30*time.Second {
			t.Errorf("RetryAfter = %v, want 30s", rle.RetryAfter)
		}
	}
}

func TestSquareConnector_Do_ValidationError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "INVALID_REQUEST_ERROR", "code": "MISSING_REQUIRED_PARAMETER", "detail": "Missing required parameter: location_id"},
			},
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

func TestSquareConnector_Do_Timeout(t *testing.T) {
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

func TestSquareConnector_Do_MissingCredentials(t *testing.T) {
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

func TestSquareConnector_Do_ForbiddenAuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "AUTHENTICATION_ERROR", "code": "FORBIDDEN", "detail": "The provided access token does not have the required permissions."},
			},
		})
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

func TestSquareConnector_Do_ForbiddenNonAuth(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "API_ERROR", "code": "FORBIDDEN", "detail": "Action not allowed."},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	// Should be ExternalError, not AuthError, since category is not AUTHENTICATION_ERROR.
	if connectors.IsAuthError(err) {
		t.Errorf("expected ExternalError for non-auth 403, got AuthError: %v", err)
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestSquareConnector_BaseURLForCreds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		conn    *SquareConnector
		creds   connectors.Credentials
		wantURL string
	}{
		{
			name:    "production by default",
			conn:    New(),
			creds:   validCreds(),
			wantURL: productionBaseURL,
		},
		{
			name:    "sandbox via creds",
			conn:    New(),
			creds:   sandboxCreds(),
			wantURL: sandboxBaseURL,
		},
		{
			name:    "test override takes precedence",
			conn:    newForTest(nil, "http://localhost:9999"),
			creds:   sandboxCreds(),
			wantURL: "http://localhost:9999",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.conn.baseURLForCreds(tt.creds)
			if got != tt.wantURL {
				t.Errorf("baseURLForCreds() = %q, want %q", got, tt.wantURL)
			}
		})
	}
}

func TestNewIdempotencyKey(t *testing.T) {
	t.Parallel()

	key1 := newIdempotencyKey()
	key2 := newIdempotencyKey()

	if key1 == "" {
		t.Error("newIdempotencyKey() returned empty string")
	}
	if key1 == key2 {
		t.Errorf("newIdempotencyKey() returned duplicate keys: %q", key1)
	}
	// UUID v4 format: 8-4-4-4-12 hex digits (xxxxxxxx-xxxx-4xxx-[89ab]xxx-xxxxxxxxxxxx)
	if len(key1) != 36 {
		t.Errorf("newIdempotencyKey() length = %d, want 36", len(key1))
	}
	// Verify version nibble (position 14) is '4' per RFC 4122 §4.4.
	if key1[14] != '4' {
		t.Errorf("newIdempotencyKey() version nibble = %c, want '4'", key1[14])
	}
	// Verify variant bits (position 19) are one of '8', '9', 'a', 'b' per RFC 4122 §4.1.1.
	variant := key1[19]
	if variant != '8' && variant != '9' && variant != 'a' && variant != 'b' {
		t.Errorf("newIdempotencyKey() variant nibble = %c, want one of '8','9','a','b'", variant)
	}
}
