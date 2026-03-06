package expedia

import (
	"context"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestExpediaConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "expedia" {
		t.Errorf("ID() = %q, want %q", got, "expedia")
	}
}

func TestExpediaConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	if actions == nil {
		t.Fatal("Actions() returned nil, want non-nil map")
	}
	if len(actions) != 6 {
		t.Errorf("Actions() has %d entries, want 6", len(actions))
	}
}

func TestExpediaConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key and secret",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "test_key", "secret": "test_secret"}),
			wantErr: false,
		},
		{
			name:    "missing api_key",
			creds:   connectors.NewCredentials(map[string]string{"secret": "test_secret"}),
			wantErr: true,
		},
		{
			name:    "missing secret",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "test_key"}),
			wantErr: true,
		},
		{
			name:    "empty api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "", "secret": "test_secret"}),
			wantErr: true,
		},
		{
			name:    "empty secret",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "test_key", "secret": ""}),
			wantErr: true,
		},
		{
			name:    "both missing",
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

func TestExpediaConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "expedia" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "expedia")
	}
	if m.Name != "Expedia Rapid" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Expedia Rapid")
	}
	if len(m.Actions) != 6 {
		t.Fatalf("Manifest().Actions has %d items, want 6", len(m.Actions))
	}
	wantActions := []string{
		"expedia.search_hotels",
		"expedia.get_hotel",
		"expedia.price_check",
		"expedia.create_booking",
		"expedia.cancel_booking",
		"expedia.get_booking",
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
	if cred.Service != "expedia" {
		t.Errorf("credential service = %q, want %q", cred.Service, "expedia")
	}
	if cred.AuthType != "api_key" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "api_key")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	// Validate the manifest passes validation.
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestExpediaConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*ExpediaConnector)(nil)
	var _ connectors.ManifestProvider = (*ExpediaConnector)(nil)
}

func TestExpediaConnector_Signature(t *testing.T) {
	t.Parallel()

	fixedTime := time.Unix(1700000000, 0)
	c := &ExpediaConnector{
		nowFunc: func() time.Time { return fixedTime },
	}

	sig, ts := c.signature("mykey", "mysecret")

	if ts != "1700000000" {
		t.Errorf("timestamp = %q, want %q", ts, "1700000000")
	}

	// Verify the signature is SHA512(api_key + secret + timestamp).
	h := sha512.New()
	h.Write([]byte("mykey"))
	h.Write([]byte("mysecret"))
	h.Write([]byte("1700000000"))
	want := hex.EncodeToString(h.Sum(nil))

	if sig != want {
		t.Errorf("signature = %q, want %q", sig, want)
	}
}

func TestExpediaConnector_Do_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header format.
		auth := r.Header.Get("Authorization")
		if !strings.HasPrefix(auth, "EAN apikey=test_api_key,signature=") {
			t.Errorf("Authorization header = %q, want prefix %q", auth, "EAN apikey=test_api_key,signature=")
		}
		if !strings.Contains(auth, ",timestamp=") {
			t.Error("Authorization header missing timestamp")
		}

		// Verify other headers.
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want %q", got, "application/json")
		}
		if got := r.Header.Get("Customer-Ip"); got != "127.0.0.1" {
			t.Errorf("Customer-Ip = %q, want %q", got, "127.0.0.1")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", "", nil, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["status"] != "ok" {
		t.Errorf("response status = %q, want %q", resp["status"], "ok")
	}
}

func TestExpediaConnector_Do_WithBody(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		if reqBody["name"] != "test" {
			t.Errorf("name = %q, want %q", reqBody["name"], "test")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "123"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]string
	err := conn.do(t.Context(), validCreds(), http.MethodPost, "/test", "", map[string]string{"name": "test"}, &resp)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
	if resp["id"] != "123" {
		t.Errorf("response id = %q, want %q", resp["id"], "123")
	}
}

func TestExpediaConnector_Do_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"type":    "request_unauthenticated",
			"message": "Invalid API key",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", "", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("do() error = %T (%v), want *connectors.AuthError", err, err)
	}
}

func TestExpediaConnector_Do_RateLimitError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]string{
			"type":    "too_many_requests",
			"message": "Rate limit exceeded",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", "", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("do() error = %T (%v), want *connectors.RateLimitError", err, err)
	}
	var rlErr *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rlErr) {
		if rlErr.RetryAfter != 30*time.Second {
			t.Errorf("RetryAfter = %v, want %v", rlErr.RetryAfter, 30*time.Second)
		}
	}
}

func TestExpediaConnector_Do_ValidationError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"type":    "invalid_input",
			"message": "checkin date is in the past",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", "", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("do() error = %T (%v), want *connectors.ValidationError", err, err)
	}
}

func TestExpediaConnector_Do_ExternalError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{
			"type":    "unknown_internal_error",
			"message": "Internal server error",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", "", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("do() error = %T (%v), want *connectors.ExternalError", err, err)
	}
}

func TestExpediaConnector_Do_CustomCustomerIP(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Customer-Ip"); got != "203.0.113.42" {
			t.Errorf("Customer-Ip = %q, want %q", got, "203.0.113.42")
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", "203.0.113.42", nil, nil)
	if err != nil {
		t.Fatalf("do() unexpected error: %v", err)
	}
}

func TestExpediaConnector_ActionsMatchManifest(t *testing.T) {
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

func TestExpediaConnector_Do_Timeout(t *testing.T) {
	t.Parallel()

	// Create a server that never responds.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Block until the request context is canceled.
		<-r.Context().Done()
	}))
	defer srv.Close()

	// Use a very short timeout to trigger the timeout path.
	shortClient := &http.Client{Timeout: 1 * time.Millisecond}
	conn := &ExpediaConnector{
		client:  shortClient,
		baseURL: srv.URL,
		nowFunc: time.Now,
	}
	err := conn.do(t.Context(), validCreds(), http.MethodGet, "/test", "", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("do() error = %T (%v), want *connectors.TimeoutError", err, err)
	}
}

func TestExpediaConnector_Do_ContextCanceled(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)

	ctx, cancel := context.WithCancel(t.Context())
	cancel() // Cancel immediately.

	err := conn.do(ctx, validCreds(), http.MethodGet, "/test", "", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	// Should be a TimeoutError (we map context.Canceled to TimeoutError).
	if !connectors.IsTimeoutError(err) {
		t.Errorf("do() error = %T (%v), want *connectors.TimeoutError", err, err)
	}
}

func TestExpediaConnector_Do_MissingCredentials(t *testing.T) {
	t.Parallel()

	conn := New()
	err := conn.do(t.Context(), connectors.NewCredentials(map[string]string{}), http.MethodGet, "/test", "", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("do() error = %T (%v), want *connectors.ValidationError", err, err)
	}
}
