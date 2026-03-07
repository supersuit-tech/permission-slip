package plaid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateLinkToken_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/link/token/create" {
			t.Errorf("path = %s, want /link/token/create", got)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", ct)
		}
		if v := r.Header.Get("Plaid-Version"); v != plaidAPIVersion {
			t.Errorf("Plaid-Version = %q, want %q", v, plaidAPIVersion)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}
		if body["client_id"] != testClientID {
			t.Errorf("client_id = %v, want %v", body["client_id"], testClientID)
		}
		if body["secret"] != testSecret {
			t.Errorf("secret = %v, want %v", body["secret"], testSecret)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"link_token": "link-sandbox-abc123",
			"expiration": "2026-03-07T12:00:00Z",
			"request_id": "req123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.create_link_token"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.create_link_token",
		Parameters:  json.RawMessage(`{"user_id":"user123","products":["transactions","balance"]}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["link_token"] != "link-sandbox-abc123" {
		t.Errorf("link_token = %v, want link-sandbox-abc123", data["link_token"])
	}
}

func TestCreateLinkToken_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["plaid.create_link_token"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing user_id", params: `{"products":["transactions"]}`},
		{name: "missing products", params: `{"user_id":"user123"}`},
		{name: "empty products", params: `{"user_id":"user123","products":[]}`},
		{name: "invalid product", params: `{"user_id":"user123","products":["invalid"]}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "plaid.create_link_token",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestCreateLinkToken_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"error_type":    "API_ERROR",
			"error_code":    "INTERNAL_SERVER_ERROR",
			"error_message": "an unexpected error occurred",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.create_link_token"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.create_link_token",
		Parameters:  json.RawMessage(`{"user_id":"user123","products":["transactions"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCreateLinkToken_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error_type":    "INVALID_INPUT",
			"error_code":    "INVALID_API_KEYS",
			"error_message": "invalid API keys",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.create_link_token"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.create_link_token",
		Parameters:  json.RawMessage(`{"user_id":"user123","products":["transactions"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCreateLinkToken_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"error_type":    "RATE_LIMIT_EXCEEDED",
			"error_code":    "RATE_LIMIT",
			"error_message": "rate limit exceeded",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.create_link_token"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "plaid.create_link_token",
		Parameters:  json.RawMessage(`{"user_id":"user123","products":["transactions"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestCreateLinkToken_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["plaid.create_link_token"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "plaid.create_link_token",
		Parameters:  json.RawMessage(`{"user_id":"user123","products":["transactions"]}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
