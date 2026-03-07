package stripe

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCancelSubscription_Immediate(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/v1/subscriptions/sub_abc123" {
			t.Errorf("path = %s, want /v1/subscriptions/sub_abc123", r.URL.Path)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":                   "sub_abc123",
			"status":               "canceled",
			"cancel_at_period_end": false,
			"canceled_at":          1700000000,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.cancel_subscription"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.cancel_subscription",
		Parameters:  json.RawMessage(`{"subscription_id":"sub_abc123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "sub_abc123" {
		t.Errorf("id = %v, want sub_abc123", data["id"])
	}
	if data["status"] != "canceled" {
		t.Errorf("status = %v, want canceled", data["status"])
	}
}

func TestCancelSubscription_AtPeriodEnd(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST (update, not delete)", r.Method)
		}
		if r.URL.Path != "/v1/subscriptions/sub_abc123" {
			t.Errorf("path = %s, want /v1/subscriptions/sub_abc123", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if got := r.FormValue("cancel_at_period_end"); got != "true" {
			t.Errorf("cancel_at_period_end = %q, want true", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":                   "sub_abc123",
			"status":               "active",
			"cancel_at_period_end": true,
			"canceled_at":          1700000000,
			"current_period_end":   1703000000,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.cancel_subscription"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.cancel_subscription",
		Parameters:  json.RawMessage(`{"subscription_id":"sub_abc123","cancel_at_period_end":true}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["cancel_at_period_end"] != true {
		t.Errorf("cancel_at_period_end = %v, want true", data["cancel_at_period_end"])
	}
}

func TestCancelSubscription_MissingSubscriptionID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.cancel_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.cancel_subscription",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCancelSubscription_InvalidProrationBehavior(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.cancel_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.cancel_subscription",
		Parameters:  json.RawMessage(`{"subscription_id":"sub_abc123","proration_behavior":"invalid"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCancelSubscription_ImmediateWithProration(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}

		// Go's ParseForm only parses request bodies for POST/PUT/PATCH.
		// For DELETE, read the body directly and parse the form values.
		// NOTE: Use t.Errorf (not t.Fatalf) here — t.Fatalf calls
		// runtime.Goexit which panics in non-test goroutines.
		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("reading body: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		vals, err := url.ParseQuery(string(bodyBytes))
		if err != nil {
			t.Errorf("parsing body as form: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		if got := vals.Get("proration_behavior"); got != "create_prorations" {
			t.Errorf("proration_behavior = %q, want create_prorations", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":     "sub_abc123",
			"status": "canceled",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.cancel_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.cancel_subscription",
		Parameters:  json.RawMessage(`{"subscription_id":"sub_abc123","proration_behavior":"create_prorations"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}
