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

func TestUpdateSubscription_TrialEndNow(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("trial_end"); got != "now" {
			t.Errorf("trial_end = %q, want now", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "sub_abc123",
			"status": "active",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.update_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.update_subscription",
		Parameters:  json.RawMessage(`{"subscription_id":"sub_abc123","trial_end":"now"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestUpdateSubscription_InvalidTrialEnd(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.update_subscription"]

	cases := []string{"tomorrow", "123abc", "0", "-1"}
	for _, tc := range cases {
		params, _ := json.Marshal(map[string]any{
			"subscription_id": "sub_abc",
			"trial_end":       tc,
		})
		_, err := action.Execute(t.Context(), connectors.ActionRequest{
			ActionType:  "stripe.update_subscription",
			Parameters:  json.RawMessage(params),
			Credentials: validCreds(),
		})
		if err == nil {
			t.Errorf("trial_end=%q: expected error, got nil", tc)
			continue
		}
		if !connectors.IsValidationError(err) {
			t.Errorf("trial_end=%q: expected ValidationError, got %T: %v", tc, err, err)
		}
	}
}

func TestUpdateSubscription_CancelAt(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("cancel_at"); got != "1893456000" {
			t.Errorf("cancel_at = %q, want 1893456000", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id":        "sub_abc123",
			"status":    "active",
			"cancel_at": 1893456000,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.update_subscription"]

	var cancelAt int64 = 1893456000
	params, _ := json.Marshal(map[string]any{
		"subscription_id": "sub_abc123",
		"cancel_at":       cancelAt,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.update_subscription",
		Parameters:  json.RawMessage(params),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["cancel_at"] != float64(1893456000) {
		t.Errorf("cancel_at = %v, want 1893456000", data["cancel_at"])
	}
}
