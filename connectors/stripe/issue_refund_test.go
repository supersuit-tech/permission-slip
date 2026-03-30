package stripe

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestIssueRefund_FullRefundByPaymentIntent(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/refunds" {
			t.Errorf("path = %s, want /v1/refunds", r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got == "" {
			t.Error("expected Idempotency-Key header (critical for refunds)")
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			http.Error(w, "bad request", http.StatusInternalServerError)
			return
		}
		if got := r.FormValue("payment_intent"); got != "pi_abc123" {
			t.Errorf("payment_intent = %q, want pi_abc123", got)
		}
		// Full refund: amount should not be set.
		if got := r.FormValue("amount"); got != "" {
			t.Errorf("amount should be empty for full refund, got %q", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":     "re_full",
			"amount": 5000,
			"status": "succeeded",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.issue_refund"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{"payment_intent_id":"pi_abc123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "re_full" {
		t.Errorf("id = %v, want re_full", data["id"])
	}
	if data["status"] != "succeeded" {
		t.Errorf("status = %v, want succeeded", data["status"])
	}
}

func TestIssueRefund_PartialRefundByCharge(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			http.Error(w, "bad request", http.StatusInternalServerError)
			return
		}
		if got := r.FormValue("charge"); got != "ch_xyz789" {
			t.Errorf("charge = %q, want ch_xyz789", got)
		}
		if got := r.FormValue("amount"); got != "500" {
			t.Errorf("amount = %q, want 500", got)
		}
		if got := r.FormValue("reason"); got != "requested_by_customer" {
			t.Errorf("reason = %q, want requested_by_customer", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":     "re_partial",
			"amount": 500,
			"status": "succeeded",
			"reason": "requested_by_customer",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.issue_refund"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{"charge_id":"ch_xyz789","amount":500,"reason":"requested_by_customer"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["amount"] != float64(500) {
		t.Errorf("amount = %v, want 500", data["amount"])
	}
}

func TestIssueRefund_MissingBothIDs(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{"reason":"duplicate"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestIssueRefund_BothIDsProvided(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{"payment_intent_id":"pi_abc","charge_id":"ch_xyz"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestIssueRefund_InvalidReason(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{"payment_intent_id":"pi_abc","reason":"bad_reason"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestIssueRefund_NegativeAmount(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{"payment_intent_id":"pi_abc","amount":-100}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestIssueRefund_IdempotencyKeyDeterministic(t *testing.T) {
	t.Parallel()

	// Use a mutex to safely collect keys from the httptest handler goroutine.
	var mu sync.Mutex
	var keys []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		keys = append(keys, r.Header.Get("Idempotency-Key"))
		mu.Unlock()
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "re_idem",
			"amount": 1000,
			"status": "succeeded",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.issue_refund"]
	params := json.RawMessage(`{"payment_intent_id":"pi_abc123","amount":1000}`)

	// Execute twice with same params — keys should be identical.
	for range 2 {
		_, err := action.Execute(t.Context(), connectors.ActionRequest{
			ActionType:  "stripe.issue_refund",
			Parameters:  params,
			Credentials: validCreds(),
		})
		if err != nil {
			t.Fatalf("Execute() unexpected error: %v", err)
		}
	}

	mu.Lock()
	defer mu.Unlock()
	if len(keys) != 2 {
		t.Fatalf("expected 2 requests, got %d", len(keys))
	}
	if keys[0] != keys[1] {
		t.Errorf("idempotency keys should match: %q vs %q", keys[0], keys[1])
	}
	if keys[0] == "" {
		t.Error("idempotency key should not be empty")
	}
}

func TestIssueRefund_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "This charge has already been refunded",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{"payment_intent_id":"pi_already_refunded"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestIssueRefund_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.issue_refund"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{"payment_intent_id":"pi_abc123"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}

func TestIssueRefund_TooManyMetadataKeys(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.issue_refund"]

	metadata := make(map[string]any, 51)
	for i := range 51 {
		metadata[fmt.Sprintf("key_%d", i)] = "value"
	}
	params, _ := json.Marshal(map[string]any{
		"payment_intent_id": "pi_abc123",
		"metadata":          metadata,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(params),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for too many metadata keys, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
