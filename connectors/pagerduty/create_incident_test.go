package pagerduty

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

func TestCreateIncident_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/incidents" {
			t.Errorf("path = %s, want /incidents", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Token token=pd_test_api_key_123" {
			t.Errorf("Authorization = %q, want %q", got, "Token token=pd_test_api_key_123")
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		incident := reqBody["incident"].(map[string]any)
		if incident["title"] != "Database down" {
			t.Errorf("title = %v, want %q", incident["title"], "Database down")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"incident": map[string]any{
				"id":     "P1234567",
				"status": "triggered",
				"title":  "Database down",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.create_incident"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.create_incident",
		Parameters:  json.RawMessage(`{"service_id":"PSERVICE1","title":"Database down","urgency":"high"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if result == nil || result.Data == nil {
		t.Fatal("Execute() returned nil result")
	}
}

func TestCreateIncident_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["pagerduty.create_incident"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing service_id", params: `{"title":"Bug"}`},
		{name: "missing title", params: `{"service_id":"PSERVICE1"}`},
		{name: "invalid urgency", params: `{"service_id":"PSERVICE1","title":"Bug","urgency":"critical"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "pagerduty.create_incident",
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

func TestCreateIncident_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Invalid API key"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.create_incident"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.create_incident",
		Parameters:  json.RawMessage(`{"service_id":"PSERVICE1","title":"Bug"}`),
		Credentials: connectors.NewCredentials(map[string]string{"api_key": "bad_token"}),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCreateIncident_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.create_incident"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "pagerduty.create_incident",
		Parameters:  json.RawMessage(`{"service_id":"PSERVICE1","title":"Bug"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestCreateIncident_FromHeader(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("From"); got != "agent@example.com" {
			t.Errorf("From = %q, want %q", got, "agent@example.com")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"incident": map[string]any{"id": "P1234567", "status": "triggered"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.create_incident"]

	creds := connectors.NewCredentials(map[string]string{
		"api_key": "pd_test_api_key_123",
		"email":   "agent@example.com",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.create_incident",
		Parameters:  json.RawMessage(`{"service_id":"PSERVICE1","title":"Bug"}`),
		Credentials: creds,
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateIncident_NoFromHeaderWithoutEmail(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("From"); got != "" {
			t.Errorf("From = %q, want empty (no email credential)", got)
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"incident": map[string]any{"id": "P1234567", "status": "triggered"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.create_incident"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.create_incident",
		Parameters:  json.RawMessage(`{"service_id":"PSERVICE1","title":"Bug"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateIncident_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"message": "Rate limit exceeded"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["pagerduty.create_incident"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "pagerduty.create_incident",
		Parameters:  json.RawMessage(`{"service_id":"PSERVICE1","title":"Bug"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 60*time.Second {
			t.Errorf("RetryAfter = %v, want 60s", rle.RetryAfter)
		}
	}
}
