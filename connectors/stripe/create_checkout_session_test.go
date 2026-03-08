package stripe

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateCheckoutSession_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/checkout/sessions" {
			t.Errorf("path = %s, want /v1/checkout/sessions", r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got == "" {
			t.Error("expected Idempotency-Key header on POST")
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("mode"); got != "subscription" {
			t.Errorf("mode = %q, want subscription", got)
		}
		if got := r.FormValue("line_items[0][price]"); got != "price_abc123" {
			t.Errorf("line_items[0][price] = %q, want price_abc123", got)
		}
		if got := r.FormValue("line_items[0][quantity]"); got != "1" {
			t.Errorf("line_items[0][quantity] = %q, want 1", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":      "cs_test_abc123",
			"url":     "https://checkout.stripe.com/pay/cs_test_abc123",
			"status":  "open",
			"mode":    "subscription",
			"created": 1709740800,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_checkout_session"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_checkout_session",
		Parameters:  json.RawMessage(`{"mode":"subscription","line_items":[{"price":"price_abc123","quantity":1}],"success_url":"https://example.com/success","cancel_url":"https://example.com/cancel"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "cs_test_abc123" {
		t.Errorf("id = %v, want cs_test_abc123", data["id"])
	}
	if data["url"] == nil || data["url"] == "" {
		t.Error("expected url in response")
	}
}

func TestCreateCheckoutSession_MissingMode(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_checkout_session"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_checkout_session",
		Parameters:  json.RawMessage(`{"line_items":[{"price":"price_abc","quantity":1}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCheckoutSession_MissingLineItems(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_checkout_session"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_checkout_session",
		Parameters:  json.RawMessage(`{"mode":"payment"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCheckoutSession_BothCustomerAndEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_checkout_session"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_checkout_session",
		Parameters:  json.RawMessage(`{"mode":"payment","line_items":[{"price":"price_abc","quantity":1}],"customer":"cus_abc","customer_email":"foo@example.com"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for conflicting customer/customer_email, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCheckoutSession_InvalidMode(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_checkout_session"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_checkout_session",
		Parameters:  json.RawMessage(`{"mode":"invalid","line_items":[{"price":"price_abc","quantity":1}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCheckoutSession_InsecureSuccessURL(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_checkout_session"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_checkout_session",
		Parameters:  json.RawMessage(`{"mode":"payment","line_items":[{"price":"price_abc","quantity":1}],"success_url":"http://example.com/success"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for http success_url, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCheckoutSession_InsecureCancelURL(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_checkout_session"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_checkout_session",
		Parameters:  json.RawMessage(`{"mode":"payment","line_items":[{"price":"price_abc","quantity":1}],"cancel_url":"http://example.com/cancel"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for http cancel_url, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCheckoutSession_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	_, err := conn.Actions()["stripe.create_checkout_session"].Execute(ctx, connectors.ActionRequest{
		ActionType:  "stripe.create_checkout_session",
		Parameters:  json.RawMessage(`{"mode":"payment","line_items":[{"price":"price_abc","quantity":1}],"success_url":"https://example.com/success"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
