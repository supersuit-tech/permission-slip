package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestIssueRefund_Success_FullRefund(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v1/refunds" {
			t.Errorf("path = %s, want /v1/refunds", r.URL.Path)
		}
		if r.Header.Get("Idempotency-Key") == "" {
			t.Error("expected Idempotency-Key header for refund")
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.FormValue("payment_intent"); got != "pi_abc123" {
			t.Errorf("payment_intent = %q, want %q", got, "pi_abc123")
		}
		// Full refund: no amount field should be present.
		if got := r.FormValue("amount"); got != "" {
			t.Errorf("amount = %q, want empty (full refund)", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "re_xyz",
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
	if data["id"] != "re_xyz" {
		t.Errorf("id = %v, want re_xyz", data["id"])
	}
	if data["status"] != "succeeded" {
		t.Errorf("status = %v, want succeeded", data["status"])
	}
}

func TestIssueRefund_PartialRefund(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if got := r.FormValue("charge"); got != "ch_abc" {
			t.Errorf("charge = %q, want %q", got, "ch_abc")
		}
		if got := r.FormValue("amount"); got != "500" {
			t.Errorf("amount = %q, want %q", got, "500")
		}
		if got := r.FormValue("reason"); got != "requested_by_customer" {
			t.Errorf("reason = %q, want %q", got, "requested_by_customer")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "re_partial",
			"amount": 500,
			"status": "succeeded",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{"charge_id":"ch_abc","amount":500,"reason":"requested_by_customer"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestIssueRefund_MissingPaymentAndCharge(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{"amount":500}`),
		Credentials: validCreds(),
	})
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
		Parameters:  json.RawMessage(`{"payment_intent_id":"pi_abc","reason":"because_i_said_so"}`),
		Credentials: validCreds(),
	})
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestIssueRefund_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{bad`),
		Credentials: validCreds(),
	})
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestIssueRefund_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"type":    "invalid_request_error",
				"message": "Charge has already been refunded",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.issue_refund",
		Parameters:  json.RawMessage(`{"payment_intent_id":"pi_abc"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
}
