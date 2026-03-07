package zapier

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestTriggerWebhook_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("expected JSON content type, got %q", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["name"] != "test" {
			t.Errorf("expected name 'test', got %v", body["name"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "success"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &triggerWebhookAction{conn: conn}

	params, _ := json.Marshal(triggerWebhookParams{
		Payload: json.RawMessage(`{"name":"test"}`),
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zapier.trigger_webhook",
		Parameters:  params,
		Credentials: credsWithURL(srv.URL + "/webhook"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["status"] != "triggered" {
		t.Errorf("expected status 'triggered', got %v", data["status"])
	}
}

func TestTriggerWebhook_WithResponse(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"result": "processed"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &triggerWebhookAction{conn: conn}

	params, _ := json.Marshal(triggerWebhookParams{
		Payload:         json.RawMessage(`{"key":"value"}`),
		WaitForResponse: true,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zapier.trigger_webhook",
		Parameters:  params,
		Credentials: credsWithURL(srv.URL),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["response"] == nil {
		t.Error("expected response data when wait_for_response is true")
	}
}

func TestTriggerWebhook_MissingPayload(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &triggerWebhookAction{conn: conn}

	params, _ := json.Marshal(map[string]any{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zapier.trigger_webhook",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing payload")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestTriggerWebhook_MissingCredentials(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &triggerWebhookAction{conn: conn}

	params, _ := json.Marshal(triggerWebhookParams{
		Payload: json.RawMessage(`{"key":"value"}`),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zapier.trigger_webhook",
		Parameters:  params,
		Credentials: connectors.NewCredentials(map[string]string{}),
	})
	if err == nil {
		t.Fatal("expected error for missing credentials")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestTriggerWebhook_WebhookGone(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusGone)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &triggerWebhookAction{conn: conn}

	params, _ := json.Marshal(triggerWebhookParams{
		Payload: json.RawMessage(`{"key":"value"}`),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zapier.trigger_webhook",
		Parameters:  params,
		Credentials: credsWithURL(srv.URL),
	})
	if err == nil {
		t.Fatal("expected error for gone webhook")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestTriggerWebhook_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &triggerWebhookAction{conn: conn}

	params, _ := json.Marshal(triggerWebhookParams{
		Payload: json.RawMessage(`{"key":"value"}`),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zapier.trigger_webhook",
		Parameters:  params,
		Credentials: credsWithURL(srv.URL),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestTriggerWebhook_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &triggerWebhookAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zapier.trigger_webhook",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestTriggerWebhook_ServerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client())
	action := &triggerWebhookAction{conn: conn}

	params, _ := json.Marshal(triggerWebhookParams{
		Payload: json.RawMessage(`{"key":"value"}`),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "zapier.trigger_webhook",
		Parameters:  params,
		Credentials: credsWithURL(srv.URL),
	})
	if err == nil {
		t.Fatal("expected error for server error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}
