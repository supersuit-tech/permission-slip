package zoom

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestZoomConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "zoom" {
		t.Errorf("expected ID 'zoom', got %q", c.ID())
	}
}

func TestZoomConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	if actions == nil {
		t.Fatal("Actions() returned nil, want non-nil map")
	}
}

func TestZoomConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "test-token"}),
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

func TestZoomConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*ZoomConnector)(nil)
}

func TestDoJSON_Success(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer test-zoom-access-token-123" {
			t.Errorf("unexpected Authorization header: %q", r.Header.Get("Authorization"))
		}
		if r.URL.Path != "/users/me" {
			t.Errorf("unexpected path: %q", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"id":"abc123","first_name":"Test","last_name":"User","email":"test@example.com"}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp struct {
		ID        string `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Email     string `json:"email"`
	}

	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/users/me", nil, &resp)
	if err != nil {
		t.Fatalf("doJSON() returned error: %v", err)
	}
	if resp.ID != "abc123" {
		t.Errorf("resp.ID = %q, want %q", resp.ID, "abc123")
	}
	if resp.Email != "test@example.com" {
		t.Errorf("resp.Email = %q, want %q", resp.Email, "test@example.com")
	}
}

func TestDoJSON_PostWithBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %q", r.Method)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("unexpected Content-Type: %q", r.Header.Get("Content-Type"))
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decoding request body: %v", err)
		}
		if body["topic"] != "Test Meeting" {
			t.Errorf("body topic = %q, want %q", body["topic"], "Test Meeting")
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		fmt.Fprint(w, `{"id":123456,"topic":"Test Meeting","join_url":"https://zoom.us/j/123456"}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	reqBody := map[string]any{"topic": "Test Meeting", "type": 2}
	var resp struct {
		ID      int    `json:"id"`
		Topic   string `json:"topic"`
		JoinURL string `json:"join_url"`
	}

	err := conn.doJSON(t.Context(), validCreds(), http.MethodPost, srv.URL+"/users/me/meetings", reqBody, &resp)
	if err != nil {
		t.Fatalf("doJSON() returned error: %v", err)
	}
	if resp.ID != 123456 {
		t.Errorf("resp.ID = %d, want %d", resp.ID, 123456)
	}
}

func TestDoJSON_NoContent(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	err := conn.doJSON(t.Context(), validCreds(), http.MethodDelete, srv.URL+"/meetings/123", nil, nil)
	if err != nil {
		t.Fatalf("doJSON() returned error: %v", err)
	}
}

func TestDoJSON_MissingToken(t *testing.T) {
	t.Parallel()
	conn := newForTest(http.DefaultClient, "http://localhost")
	err := conn.doJSON(t.Context(), connectors.NewCredentials(map[string]string{}), http.MethodGet, "http://localhost/test", nil, nil)
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCheckResponse_AuthError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		fmt.Fprint(w, `{"code":124,"message":"Invalid access token."}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/users/me", nil, &resp)
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_Forbidden(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, `{"code":200,"message":"Insufficient permissions."}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/users/me", nil, &resp)
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError for 403, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"code":429,"message":"Rate limit exceeded."}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/users/me/meetings", nil, &resp)
	if !connectors.IsRateLimitError(err) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 30*time.Second {
			t.Errorf("RetryAfter = %v, want 30s", rle.RetryAfter)
		}
	}
}

func TestCheckResponse_BadRequest(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `{"code":300,"message":"Invalid parameter: start_time"}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodPost, srv.URL+"/users/me/meetings", map[string]any{}, &resp)
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 400, got %T: %v", err, err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		fmt.Fprint(w, `{"code":3001,"message":"Meeting does not exist."}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/meetings/999", nil, &resp)
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, `{"code":500,"message":"Internal server error"}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/users/me", nil, &resp)
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError for 500, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimitNoRetryAfter(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"code":429,"message":"Rate limit exceeded."}`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/users/me", nil, &resp)
	if !connectors.IsRateLimitError(err) {
		t.Fatalf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != defaultRetryAfter {
			t.Errorf("RetryAfter = %v, want default %v", rle.RetryAfter, defaultRetryAfter)
		}
	}
}

func TestCheckResponse_MalformedErrorBody(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprint(w, `not json`)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	var resp map[string]any
	err := conn.doJSON(t.Context(), validCreds(), http.MethodGet, srv.URL+"/test", nil, &resp)
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 400 with malformed body, got %T: %v", err, err)
	}
}
