package stripe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
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

func TestCreatePrice_WithTaxBehavior(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("tax_behavior"); got != "exclusive" {
			t.Errorf("tax_behavior = %q, want exclusive", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id":           "price_abc123",
			"currency":     "usd",
			"product":      "prod_abc123",
			"unit_amount":  2000,
			"tax_behavior": "exclusive",
			"active":       true,
			"created":      1709740800,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_price"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_price",
		Parameters:  json.RawMessage(`{"currency":"usd","product":"prod_abc123","unit_amount":2000,"tax_behavior":"exclusive"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["tax_behavior"] != "exclusive" {
		t.Errorf("tax_behavior = %v, want exclusive", data["tax_behavior"])
	}
}

func TestCreatePrice_InvalidTaxBehavior(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_price"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_price",
		Parameters:  json.RawMessage(`{"currency":"usd","product":"prod_abc","unit_amount":1000,"tax_behavior":"automatic"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for invalid tax_behavior, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreatePrice_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	_, err := conn.Actions()["stripe.create_price"].Execute(ctx, connectors.ActionRequest{
		ActionType:  "stripe.create_price",
		Parameters:  json.RawMessage(`{"currency":"usd","product":"prod_abc","unit_amount":2000}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
