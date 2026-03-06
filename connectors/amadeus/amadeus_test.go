package amadeus

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAmadeusConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "amadeus" {
		t.Errorf("ID() = %q, want %q", got, "amadeus")
	}
}

func TestAmadeusConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	if len(actions) != 0 {
		t.Errorf("Actions() returned %d actions, want 0 (Phase 1 scaffold)", len(actions))
	}
}

func TestAmadeusConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid credentials",
			creds:   connectors.NewCredentials(map[string]string{"client_id": "id123", "client_secret": "secret456"}),
			wantErr: false,
		},
		{
			name:    "missing client_id",
			creds:   connectors.NewCredentials(map[string]string{"client_secret": "secret456"}),
			wantErr: true,
		},
		{
			name:    "empty client_id",
			creds:   connectors.NewCredentials(map[string]string{"client_id": "", "client_secret": "secret456"}),
			wantErr: true,
		},
		{
			name:    "missing client_secret",
			creds:   connectors.NewCredentials(map[string]string{"client_id": "id123"}),
			wantErr: true,
		},
		{
			name:    "empty client_secret",
			creds:   connectors.NewCredentials(map[string]string{"client_id": "id123", "client_secret": ""}),
			wantErr: true,
		},
		{
			name:    "empty credentials",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "wrong key names",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "key123"}),
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

func TestAmadeusConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "amadeus" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "amadeus")
	}
	if m.Name != "Amadeus" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Amadeus")
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}
	cred := m.RequiredCredentials[0]
	if cred.Service != "amadeus" {
		t.Errorf("credential service = %q, want %q", cred.Service, "amadeus")
	}
	if cred.AuthType != "api_key" {
		t.Errorf("credential auth_type = %q, want %q", cred.AuthType, "api_key")
	}
	if cred.InstructionsURL == "" {
		t.Error("credential instructions_url is empty, want a URL")
	}

	// Validate() is skipped in Phase 1 because the manifest has no actions
	// yet. Phase 2 adds actions and re-enables this check.
}

func TestAmadeusConnector_ActionsMatchManifest(t *testing.T) {
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

func TestAmadeusConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*AmadeusConnector)(nil)
	var _ connectors.ManifestProvider = (*AmadeusConnector)(nil)
}

func TestAmadeusConnector_EnsureToken_Success(t *testing.T) {
	t.Parallel()

	srv := newTestServer(nil)
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	token, err := c.ensureToken(t.Context(), validCreds())
	if err != nil {
		t.Fatalf("ensureToken() error = %v", err)
	}
	if token != "test-access-token-123" {
		t.Errorf("ensureToken() = %q, want %q", token, "test-access-token-123")
	}
}

func TestAmadeusConnector_EnsureToken_Caching(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/security/oauth2/token" {
			callCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(tokenJSON))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)

	// First call should hit the server.
	_, err := c.ensureToken(t.Context(), validCreds())
	if err != nil {
		t.Fatalf("first ensureToken() error = %v", err)
	}
	if got := callCount.Load(); got != 1 {
		t.Fatalf("expected 1 token request, got %d", got)
	}

	// Second call should use cached token.
	_, err = c.ensureToken(t.Context(), validCreds())
	if err != nil {
		t.Fatalf("second ensureToken() error = %v", err)
	}
	if got := callCount.Load(); got != 1 {
		t.Errorf("expected 1 token request (cached), got %d", got)
	}
}

func TestAmadeusConnector_EnsureToken_Refresh(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/security/oauth2/token" {
			callCount.Add(1)
			w.Header().Set("Content-Type", "application/json")
			// Token that expires in 30 seconds (less than the 60s buffer).
			_, _ = w.Write([]byte(`{
				"access_token": "refreshed-token",
				"expires_in": 30
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)

	// First call.
	_, err := c.ensureToken(t.Context(), validCreds())
	if err != nil {
		t.Fatalf("first ensureToken() error = %v", err)
	}

	// Second call should refresh because token is within the buffer window.
	_, err = c.ensureToken(t.Context(), validCreds())
	if err != nil {
		t.Fatalf("second ensureToken() error = %v", err)
	}
	if got := callCount.Load(); got != 2 {
		t.Errorf("expected 2 token requests (refresh), got %d", got)
	}
}

func TestAmadeusConnector_EnsureToken_AuthError(t *testing.T) {
	t.Parallel()

	srv := newTestServerWithTokenError(http.StatusUnauthorized,
		`{"error": "unauthorized", "error_description": "Invalid client credentials"}`)
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	_, err := c.ensureToken(t.Context(), validCreds())
	if err == nil {
		t.Fatal("ensureToken() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("ensureToken() returned %T, want *connectors.AuthError", err)
	}
}

func TestAmadeusConnector_Do_Success(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		// Verify auth header uses the token.
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test-access-token-123")
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want %q", got, "application/json")
		}
		if r.URL.Path != "/v1/test/endpoint" {
			t.Errorf("path = %q, want %q", r.URL.Path, "/v1/test/endpoint")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result": "ok"}`))
	})
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	var resp struct {
		Result string `json:"result"`
	}
	err := c.do(t.Context(), validCreds(), http.MethodGet, "/v1/test/endpoint", nil, &resp)
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}
	if resp.Result != "ok" {
		t.Errorf("do() result = %q, want %q", resp.Result, "ok")
	}
}

func TestAmadeusConnector_Do_WithRequestBody(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %q, want %q", r.Method, http.MethodPost)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want %q", ct, "application/json")
		}

		body, _ := io.ReadAll(r.Body)
		var payload map[string]string
		if err := json.Unmarshal(body, &payload); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if payload["key"] != "value" {
			t.Errorf("request body key = %q, want %q", payload["key"], "value")
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"created": true}`))
	})
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	reqBody := map[string]string{"key": "value"}
	var resp struct {
		Created bool `json:"created"`
	}
	err := c.do(t.Context(), validCreds(), http.MethodPost, "/v1/test/create", reqBody, &resp)
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}
	if !resp.Created {
		t.Error("do() resp.Created = false, want true")
	}
}

func TestAmadeusConnector_Do_RateLimitError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "5")
		w.WriteHeader(http.StatusTooManyRequests)
		_, _ = w.Write(amadeusErrorResponse(429, 38194, "Too many requests", "Rate limit exceeded"))
	})
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	err := c.do(t.Context(), validCreds(), http.MethodGet, "/v1/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("do() returned %T, want *connectors.RateLimitError", err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 5*time.Second {
			t.Errorf("RetryAfter = %v, want %v", rle.RetryAfter, 5*time.Second)
		}
	}
}

func TestAmadeusConnector_Do_AuthError(t *testing.T) {
	t.Parallel()

	// do() retries once on 401 after invalidating the token cache.
	// Both attempts should fail with 401.
	var apiCalls atomic.Int32
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		apiCalls.Add(1)
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write(amadeusErrorResponse(401, 38190, "Unauthorized", "Invalid access token"))
	})
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	err := c.do(t.Context(), validCreds(), http.MethodGet, "/v1/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("do() returned %T, want *connectors.AuthError", err)
	}
	// Should have retried once (2 API calls total).
	if got := apiCalls.Load(); got != 2 {
		t.Errorf("expected 2 API calls (retry on 401), got %d", got)
	}
}

func TestAmadeusConnector_Do_AuthRetrySuccess(t *testing.T) {
	t.Parallel()

	// First API call returns 401 (expired token), second succeeds after
	// token refresh.
	var apiCalls atomic.Int32
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		n := apiCalls.Add(1)
		if n == 1 {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write(amadeusErrorResponse(401, 38190, "Unauthorized", "Token expired"))
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok": true}`))
	})
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	var resp struct {
		OK bool `json:"ok"`
	}
	err := c.do(t.Context(), validCreds(), http.MethodGet, "/v1/test", nil, &resp)
	if err != nil {
		t.Fatalf("do() error = %v", err)
	}
	if !resp.OK {
		t.Error("do() resp.OK = false, want true")
	}
	if got := apiCalls.Load(); got != 2 {
		t.Errorf("expected 2 API calls (retry on 401), got %d", got)
	}
}

func TestAmadeusConnector_Do_ValidationError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write(amadeusErrorResponse(400, 477, "INVALID FORMAT", "Origin must be an IATA code"))
	})
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	err := c.do(t.Context(), validCreds(), http.MethodGet, "/v1/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("do() returned %T, want *connectors.ValidationError", err)
	}
}

func TestAmadeusConnector_Do_ExternalError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write(amadeusErrorResponse(500, 141, "SYSTEM ERROR", "Internal server error"))
	})
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	err := c.do(t.Context(), validCreds(), http.MethodGet, "/v1/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("do() returned %T, want *connectors.ExternalError", err)
	}
}

func TestAmadeusConnector_Do_ForbiddenError_NoRetry(t *testing.T) {
	t.Parallel()

	// 403 (Forbidden) should NOT trigger a retry — it means the credentials
	// lack permission, not that the token expired.
	var apiCalls atomic.Int32
	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		apiCalls.Add(1)
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write(amadeusErrorResponse(403, 38196, "Forbidden", "Access denied"))
	})
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	err := c.do(t.Context(), validCreds(), http.MethodGet, "/v1/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("do() returned %T, want *connectors.AuthError", err)
	}
	// Should NOT have retried (only 1 API call).
	if got := apiCalls.Load(); got != 1 {
		t.Errorf("expected 1 API call (no retry on 403), got %d", got)
	}
}

func TestAmadeusConnector_Do_NotFoundError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write(amadeusErrorResponse(404, 1797, "NOT FOUND", "No hotel found for the given ID"))
	})
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)
	err := c.do(t.Context(), validCreds(), http.MethodGet, "/v1/test", nil, nil)
	if err == nil {
		t.Fatal("do() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("do() returned %T, want *connectors.ValidationError", err)
	}
}

func TestAmadeusConnector_EnsureToken_PerClientID(t *testing.T) {
	t.Parallel()

	// Two different client_ids should get separate cached tokens.
	var tokenCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/security/oauth2/token" {
			tokenCount.Add(1)
			_ = r.ParseForm()
			clientID := r.FormValue("client_id")
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"access_token": "token-for-` + clientID + `",
				"expires_in": 1799
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := newForTest(srv.Client(), srv.URL)

	credsA := connectors.NewCredentials(map[string]string{
		"client_id": "client-a", "client_secret": "secret-a",
	})
	credsB := connectors.NewCredentials(map[string]string{
		"client_id": "client-b", "client_secret": "secret-b",
	})

	tokenA, err := c.ensureToken(t.Context(), credsA)
	if err != nil {
		t.Fatalf("ensureToken(A) error = %v", err)
	}
	if tokenA != "token-for-client-a" {
		t.Errorf("tokenA = %q, want %q", tokenA, "token-for-client-a")
	}

	tokenB, err := c.ensureToken(t.Context(), credsB)
	if err != nil {
		t.Fatalf("ensureToken(B) error = %v", err)
	}
	if tokenB != "token-for-client-b" {
		t.Errorf("tokenB = %q, want %q", tokenB, "token-for-client-b")
	}

	if got := tokenCount.Load(); got != 2 {
		t.Errorf("expected 2 token requests (one per client_id), got %d", got)
	}

	// Calling again with credsA should use cached token (no new request).
	_, err = c.ensureToken(t.Context(), credsA)
	if err != nil {
		t.Fatalf("ensureToken(A again) error = %v", err)
	}
	if got := tokenCount.Load(); got != 2 {
		t.Errorf("expected 2 token requests (cached), got %d", got)
	}
}

func TestAmadeusConnector_ResolveBaseURL(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name string
		env  string
		want string
	}{
		{"empty defaults to test", "", defaultTestBaseURL},
		{"test environment", "test", defaultTestBaseURL},
		{"production environment", "production", defaultProductionBaseURL},
		{"unknown defaults to test", "staging", defaultTestBaseURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			creds := connectors.NewCredentials(map[string]string{
				"client_id":     "id",
				"client_secret": "secret",
				"environment":   tt.env,
			})
			got := c.resolveBaseURL(creds)
			if got != tt.want {
				t.Errorf("resolveBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	if err := checkResponse(200, nil, nil); err != nil {
		t.Errorf("checkResponse(200) = %v, want nil", err)
	}
}

func TestExtractErrorMessage_WithDetail(t *testing.T) {
	t.Parallel()
	body := amadeusErrorResponse(400, 477, "INVALID FORMAT", "Origin must be an IATA code")
	msg := extractErrorMessage(body)
	if msg != "Origin must be an IATA code" {
		t.Errorf("extractErrorMessage() = %q, want %q", msg, "Origin must be an IATA code")
	}
}

func TestExtractErrorMessage_TitleFallback(t *testing.T) {
	t.Parallel()
	body := amadeusErrorResponse(500, 141, "SYSTEM ERROR", "")
	msg := extractErrorMessage(body)
	if msg != "SYSTEM ERROR" {
		t.Errorf("extractErrorMessage() = %q, want %q", msg, "SYSTEM ERROR")
	}
}

func TestExtractErrorMessage_RawBodyFallback(t *testing.T) {
	t.Parallel()
	msg := extractErrorMessage([]byte("not json"))
	if msg != "not json" {
		t.Errorf("extractErrorMessage() = %q, want %q", msg, "not json")
	}
}
