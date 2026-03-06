package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreatePaymentLink_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/payment_links" {
			t.Errorf("path = %s, want /v1/payment_links", r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Error("expected Idempotency-Key header")
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.FormValue("line_items[0][price]"); got != "price_abc" {
			t.Errorf("line_items[0][price] = %q, want %q", got, "price_abc")
		}
		if got := r.FormValue("line_items[0][quantity]"); got != "2" {
			t.Errorf("line_items[0][quantity] = %q, want %q", got, "2")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":  "plink_abc",
			"url": "https://buy.stripe.com/test_abc",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_payment_link"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_payment_link",
		Parameters:  json.RawMessage(`{"line_items":[{"price_id":"price_abc","quantity":2}]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "plink_abc" {
		t.Errorf("id = %v, want plink_abc", data["id"])
	}
	if data["url"] != "https://buy.stripe.com/test_abc" {
		t.Errorf("url = %v, want https://buy.stripe.com/test_abc", data["url"])
	}
}

func TestCreatePaymentLink_DefaultQuantity(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.FormValue("line_items[0][quantity]"); got != "1" {
			t.Errorf("line_items[0][quantity] = %q, want %q (default)", got, "1")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":  "plink_default",
			"url": "https://buy.stripe.com/test_default",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_payment_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_payment_link",
		Parameters:  json.RawMessage(`{"line_items":[{"price_id":"price_abc"}]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreatePaymentLink_WithRedirect(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.FormValue("after_completion[type]"); got != "redirect" {
			t.Errorf("after_completion[type] = %q, want %q", got, "redirect")
		}
		if got := r.FormValue("after_completion[redirect][url]"); got != "https://example.com/thanks" {
			t.Errorf("after_completion[redirect][url] = %q, want %q", got, "https://example.com/thanks")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":  "plink_redirect",
			"url": "https://buy.stripe.com/test_redirect",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_payment_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_payment_link",
		Parameters:  json.RawMessage(`{"line_items":[{"price_id":"price_abc"}],"after_completion":"https://example.com/thanks"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreatePaymentLink_MissingLineItems(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_payment_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_payment_link",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreatePaymentLink_MissingPriceID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_payment_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_payment_link",
		Parameters:  json.RawMessage(`{"line_items":[{"quantity":1}]}`),
		Credentials: validCreds(),
	})
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreatePaymentLink_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_payment_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_payment_link",
		Parameters:  json.RawMessage(`{bad`),
		Credentials: validCreds(),
	})
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
