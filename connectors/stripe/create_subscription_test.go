package stripe

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateSubscription_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/subscriptions" {
			t.Errorf("path = %s, want /v1/subscriptions", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			http.Error(w, "bad request", http.StatusInternalServerError)
			return
		}
		if got := r.FormValue("customer"); got != "cus_abc123" {
			t.Errorf("customer = %q, want cus_abc123", got)
		}
		if got := r.FormValue("items[0][price]"); got != "price_xyz" {
			t.Errorf("items[0][price] = %q, want price_xyz", got)
		}
		if got := r.FormValue("trial_period_days"); got != "14" {
			t.Errorf("trial_period_days = %q, want 14", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":       "sub_new123",
			"status":   "trialing",
			"customer": "cus_abc123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_subscription"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_subscription",
		Parameters:  json.RawMessage(`{"customer":"cus_abc123","items":[{"price":"price_xyz"}],"trial_period_days":14}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "sub_new123" {
		t.Errorf("id = %v, want sub_new123", data["id"])
	}
	if data["status"] != "trialing" {
		t.Errorf("status = %v, want trialing", data["status"])
	}
}

func TestCreateSubscription_MissingCustomer(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_subscription",
		Parameters:  json.RawMessage(`{"items":[{"price":"price_xyz"}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateSubscription_MissingItems(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_subscription",
		Parameters:  json.RawMessage(`{"customer":"cus_abc123"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateSubscription_EmptyItemPrice(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_subscription",
		Parameters:  json.RawMessage(`{"customer":"cus_abc123","items":[{"price":""}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateSubscription_InvalidPaymentBehavior(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_subscription",
		Parameters:  json.RawMessage(`{"customer":"cus_abc123","items":[{"price":"price_xyz"}],"payment_behavior":"bad_value"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateSubscription_NegativeTrialDays(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_subscription",
		Parameters:  json.RawMessage(`{"customer":"cus_abc123","items":[{"price":"price_xyz"}],"trial_period_days":-5}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateSubscription_NegativeQuantity(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_subscription",
		Parameters:  json.RawMessage(`{"customer":"cus_abc123","items":[{"price":"price_xyz","quantity":-1}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateSubscription_MultipleItemsWithQuantity(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			http.Error(w, "bad request", http.StatusInternalServerError)
			return
		}
		if got := r.FormValue("items[0][price]"); got != "price_a" {
			t.Errorf("items[0][price] = %q, want price_a", got)
		}
		if got := r.FormValue("items[0][quantity]"); got != "2" {
			t.Errorf("items[0][quantity] = %q, want 2", got)
		}
		if got := r.FormValue("items[1][price]"); got != "price_b" {
			t.Errorf("items[1][price] = %q, want price_b", got)
		}
		if got := r.FormValue("items[1][quantity]"); got != "5" {
			t.Errorf("items[1][quantity] = %q, want 5", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":       "sub_multi",
			"status":   "active",
			"customer": "cus_abc123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_subscription"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_subscription",
		Parameters:  json.RawMessage(`{"customer":"cus_abc123","items":[{"price":"price_a","quantity":2},{"price":"price_b","quantity":5}]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateSubscription_TooManyItems(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_subscription"]

	items := make([]map[string]any, 21)
	for i := range items {
		items[i] = map[string]any{"price": fmt.Sprintf("price_%d", i)}
	}
	params, _ := json.Marshal(map[string]any{
		"customer": "cus_abc123",
		"items":    items,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_subscription",
		Parameters:  json.RawMessage(params),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
