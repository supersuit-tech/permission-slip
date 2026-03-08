package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreatePrice_RecurringSuccess(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/prices" {
			t.Errorf("path = %s, want /v1/prices", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("currency"); got != "usd" {
			t.Errorf("currency = %q, want usd", got)
		}
		if got := r.FormValue("product"); got != "prod_abc123" {
			t.Errorf("product = %q, want prod_abc123", got)
		}
		if got := r.FormValue("unit_amount"); got != "2000" {
			t.Errorf("unit_amount = %q, want 2000", got)
		}
		if got := r.FormValue("recurring[interval]"); got != "month" {
			t.Errorf("recurring[interval] = %q, want month", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":          "price_abc123",
			"currency":    "usd",
			"product":     "prod_abc123",
			"unit_amount": 2000,
			"type":        "recurring",
			"active":      true,
			"created":     1709740800,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_price"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_price",
		Parameters:  json.RawMessage(`{"currency":"usd","product":"prod_abc123","unit_amount":2000,"recurring":{"interval":"month"}}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "price_abc123" {
		t.Errorf("id = %v, want price_abc123", data["id"])
	}
}

func TestCreatePrice_MissingRequired(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_price"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_price",
		Parameters:  json.RawMessage(`{"currency":"usd"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreatePrice_InvalidCurrency(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_price"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_price",
		Parameters:  json.RawMessage(`{"currency":"dollars","product":"prod_abc","unit_amount":1000}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreatePrice_InvalidRecurringInterval(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_price"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_price",
		Parameters:  json.RawMessage(`{"currency":"usd","product":"prod_abc","unit_amount":1000,"recurring":{"interval":"biweekly"}}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
