package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestInitiatePayout_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/payouts" {
			t.Errorf("path = %s, want /v1/payouts", r.URL.Path)
		}
		if got := r.Header.Get("Idempotency-Key"); got == "" {
			t.Error("expected Idempotency-Key header (critical for payouts)")
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if got := r.FormValue("amount"); got != "10000" {
			t.Errorf("amount = %q, want 10000", got)
		}
		if got := r.FormValue("currency"); got != "usd" {
			t.Errorf("currency = %q, want usd", got)
		}
		if got := r.FormValue("description"); got != "Monthly payout" {
			t.Errorf("description = %q, want Monthly payout", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":           "po_abc123",
			"amount":       10000,
			"currency":     "usd",
			"status":       "pending",
			"arrival_date": 1700100000,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.initiate_payout"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.initiate_payout",
		Parameters:  json.RawMessage(`{"amount":10000,"currency":"usd","description":"Monthly payout"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "po_abc123" {
		t.Errorf("id = %v, want po_abc123", data["id"])
	}
	if data["status"] != "pending" {
		t.Errorf("status = %v, want pending", data["status"])
	}
	if data["amount"] != float64(10000) {
		t.Errorf("amount = %v, want 10000", data["amount"])
	}
}

func TestInitiatePayout_WithDestination(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if got := r.FormValue("destination"); got != "ba_abc123" {
			t.Errorf("destination = %q, want ba_abc123", got)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"id":          "po_dest123",
			"amount":      5000,
			"currency":    "usd",
			"status":      "pending",
			"destination": "ba_abc123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.initiate_payout"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.initiate_payout",
		Parameters:  json.RawMessage(`{"amount":5000,"currency":"usd","destination":"ba_abc123"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestInitiatePayout_MissingAmount(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.initiate_payout"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.initiate_payout",
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

func TestInitiatePayout_NegativeAmount(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.initiate_payout"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.initiate_payout",
		Parameters:  json.RawMessage(`{"amount":-100,"currency":"usd"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestInitiatePayout_MissingCurrency(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.initiate_payout"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.initiate_payout",
		Parameters:  json.RawMessage(`{"amount":10000}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestInitiatePayout_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "Insufficient funds in Stripe account",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.initiate_payout"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.initiate_payout",
		Parameters:  json.RawMessage(`{"amount":10000,"currency":"usd"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
