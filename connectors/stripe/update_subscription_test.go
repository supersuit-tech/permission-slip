package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateSubscription_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/subscriptions/sub_abc123" {
			t.Errorf("path = %s, want /v1/subscriptions/sub_abc123", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("items[0][id]"); got != "si_abc123" {
			t.Errorf("items[0][id] = %q, want si_abc123", got)
		}
		if got := r.FormValue("items[0][price]"); got != "price_new123" {
			t.Errorf("items[0][price] = %q, want price_new123", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":       "sub_abc123",
			"status":   "active",
			"customer": "cus_abc123",
			"created":  1709740800,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.update_subscription"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.update_subscription",
		Parameters:  json.RawMessage(`{"subscription_id":"sub_abc123","items":[{"id":"si_abc123","price":"price_new123"}],"proration_behavior":"create_prorations"}`),
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
}

func TestUpdateSubscription_MissingSubscriptionID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.update_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.update_subscription",
		Parameters:  json.RawMessage(`{"items":[{"price":"price_abc"}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateSubscription_InvalidProrationBehavior(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.update_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.update_subscription",
		Parameters:  json.RawMessage(`{"subscription_id":"sub_abc","proration_behavior":"invalid"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateSubscription_ItemMissingIDAndPrice(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.update_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.update_subscription",
		Parameters:  json.RawMessage(`{"subscription_id":"sub_abc","items":[{"quantity":2}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for item missing id and price, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
