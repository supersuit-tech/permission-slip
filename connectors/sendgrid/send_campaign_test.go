package sendgrid

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSendCampaign_Success(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.Header.Get("Authorization"); got != "Bearer "+testAPIKey {
			t.Errorf("Authorization = %q, want Bearer %s", got, testAPIKey)
		}

		callCount++
		switch callCount {
		case 1:
			// Create single send
			if r.Method != http.MethodPost || r.URL.Path != "/marketing/singlesends" {
				t.Errorf("call 1: %s %s, want POST /marketing/singlesends", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]any{"id": "ss_123"})
		case 2:
			// Schedule (send now)
			if r.Method != http.MethodPut || r.URL.Path != "/marketing/singlesends/ss_123/schedule" {
				t.Errorf("call 2: %s %s, want PUT /marketing/singlesends/ss_123/schedule", r.Method, r.URL.Path)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"status": "scheduled"})
		default:
			t.Errorf("unexpected call %d: %s %s", callCount, r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.send_campaign"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "sendgrid.send_campaign",
		Parameters: json.RawMessage(`{
			"name": "March Newsletter",
			"subject": "Hello World",
			"html_content": "<h1>Hi</h1>",
			"list_ids": ["list_abc"],
			"sender_id": 42
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["singlesend_id"] != "ss_123" {
		t.Errorf("singlesend_id = %v, want ss_123", data["singlesend_id"])
	}
	if data["status"] != "scheduled" {
		t.Errorf("status = %v, want scheduled", data["status"])
	}
}

func TestSendCampaign_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.send_campaign"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing name", params: `{"subject":"Hi","list_ids":["a"],"sender_id":1,"html_content":"<p>hi</p>"}`},
		{name: "missing subject", params: `{"name":"Test","list_ids":["a"],"sender_id":1,"html_content":"<p>hi</p>"}`},
		{name: "missing list_ids", params: `{"name":"Test","subject":"Hi","sender_id":1,"html_content":"<p>hi</p>"}`},
		{name: "empty list_ids", params: `{"name":"Test","subject":"Hi","list_ids":[],"sender_id":1,"html_content":"<p>hi</p>"}`},
		{name: "missing sender_id", params: `{"name":"Test","subject":"Hi","list_ids":["a"],"html_content":"<p>hi</p>"}`},
		{name: "missing content", params: `{"name":"Test","subject":"Hi","list_ids":["a"],"sender_id":1}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "sendgrid.send_campaign",
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

func TestSendCampaign_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"message": "Internal error"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.send_campaign"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "sendgrid.send_campaign",
		Parameters: json.RawMessage(`{
			"name":"Test","subject":"Hi","list_ids":["a"],"sender_id":1,"html_content":"<p>hi</p>"
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestSendCampaign_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"message": "authorization required"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.send_campaign"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "sendgrid.send_campaign",
		Parameters: json.RawMessage(`{
			"name":"Test","subject":"Hi","list_ids":["a"],"sender_id":1,"html_content":"<p>hi</p>"
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestSendCampaign_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{{"message": "Too many requests"}},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.send_campaign"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "sendgrid.send_campaign",
		Parameters: json.RawMessage(`{
			"name":"Test","subject":"Hi","list_ids":["a"],"sender_id":1,"html_content":"<p>hi</p>"
		}`),
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
		if rle.RetryAfter != 30*time.Second {
			t.Errorf("RetryAfter = %v, want 30s", rle.RetryAfter)
		}
	}
}

func TestSendCampaign_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.send_campaign"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType: "sendgrid.send_campaign",
		Parameters: json.RawMessage(`{
			"name":"Test","subject":"Hi","list_ids":["a"],"sender_id":1,"html_content":"<p>hi</p>"
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
