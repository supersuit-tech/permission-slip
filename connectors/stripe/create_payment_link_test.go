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

func TestCreatePaymentLink_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/payment_links" {
			t.Errorf("path = %s, want /v1/payment_links", r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got == "" {
			t.Error("expected Idempotency-Key header on POST")
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("line_items[0][price]"); got != "price_abc" {
			t.Errorf("line_items[0][price] = %q, want price_abc", got)
		}
		if got := r.FormValue("line_items[0][quantity]"); got != "2" {
			t.Errorf("line_items[0][quantity] = %q, want 2", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":     "plink_test123",
			"url":    "https://buy.stripe.com/test_abc",
			"active": true,
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
	if data["id"] != "plink_test123" {
		t.Errorf("id = %v, want plink_test123", data["id"])
	}
	if data["url"] != "https://buy.stripe.com/test_abc" {
		t.Errorf("url = %v, want https://buy.stripe.com/test_abc", data["url"])
	}
	if data["active"] != true {
		t.Errorf("active = %v, want true", data["active"])
	}
}

func TestCreatePaymentLink_WithRedirect(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("after_completion[type]"); got != "redirect" {
			t.Errorf("after_completion[type] = %q, want redirect", got)
		}
		if got := r.FormValue("after_completion[redirect][url]"); got != "https://example.com/thanks" {
			t.Errorf("after_completion[redirect][url] = %q, want https://example.com/thanks", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":  "plink_redirect",
			"url": "https://buy.stripe.com/redirect",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_payment_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_payment_link",
		Parameters:  json.RawMessage(`{"line_items":[{"price_id":"price_abc","quantity":1}],"after_completion":"https://example.com/thanks"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreatePaymentLink_WithPromotionCodes(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("allow_promotion_codes"); got != "true" {
			t.Errorf("allow_promotion_codes = %q, want true", got)
		}
		json.NewEncoder(w).Encode(map[string]any{
			"id":  "plink_promo",
			"url": "https://buy.stripe.com/promo",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_payment_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_payment_link",
		Parameters:  json.RawMessage(`{"line_items":[{"price_id":"price_abc","quantity":1}],"allow_promotion_codes":true}`),
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
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
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
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreatePaymentLink_InvalidQuantity(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_payment_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_payment_link",
		Parameters:  json.RawMessage(`{"line_items":[{"price_id":"price_abc","quantity":0}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreatePaymentLink_MultipleLineItems(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.FormValue("line_items[0][price]"); got != "price_a" {
			t.Errorf("line_items[0][price] = %q, want price_a", got)
		}
		if got := r.FormValue("line_items[1][price]"); got != "price_b" {
			t.Errorf("line_items[1][price] = %q, want price_b", got)
		}
		if got := r.FormValue("line_items[1][quantity]"); got != "3" {
			t.Errorf("line_items[1][quantity] = %q, want 3", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":  "plink_multi",
			"url": "https://buy.stripe.com/multi",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_payment_link"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "stripe.create_payment_link",
		Parameters: json.RawMessage(`{
			"line_items": [
				{"price_id": "price_a", "quantity": 1},
				{"price_id": "price_b", "quantity": 3}
			]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreatePaymentLink_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_payment_link"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "stripe.create_payment_link",
		Parameters:  json.RawMessage(`{"line_items":[{"price_id":"price_abc","quantity":1}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
