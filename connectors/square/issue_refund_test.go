package square

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestIssueRefund_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/refunds" {
			t.Errorf("path = %s, want /refunds", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		if _, ok := reqBody["idempotency_key"]; !ok {
			t.Error("missing idempotency_key in request body")
		}

		var paymentID string
		json.Unmarshal(reqBody["payment_id"], &paymentID)
		if paymentID != "PMT123" {
			t.Errorf("payment_id = %q, want %q", paymentID, "PMT123")
		}

		var amountMoney money
		json.Unmarshal(reqBody["amount_money"], &amountMoney)
		if amountMoney.Amount != 500 {
			t.Errorf("amount = %d, want 500", amountMoney.Amount)
		}
		if amountMoney.Currency != "USD" {
			t.Errorf("currency = %q, want USD", amountMoney.Currency)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"refund": map[string]any{
				"id":           "REF123",
				"payment_id":   "PMT123",
				"status":       "COMPLETED",
				"amount_money": map[string]any{"amount": 500, "currency": "USD"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.issue_refund"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.issue_refund",
		Parameters:  json.RawMessage(`{"payment_id": "PMT123", "amount_money": {"amount": 500, "currency": "USD"}}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "REF123" {
		t.Errorf("refund id = %v, want REF123", data["id"])
	}
	if data["status"] != "COMPLETED" {
		t.Errorf("refund status = %v, want COMPLETED", data["status"])
	}
}

func TestIssueRefund_FullRefund(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		// amount_money should not be present for full refund.
		if _, ok := reqBody["amount_money"]; ok {
			t.Error("amount_money should not be present for full refund")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"refund": map[string]any{
				"id":     "REF456",
				"status": "COMPLETED",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.issue_refund",
		Parameters:  json.RawMessage(`{"payment_id": "PMT123"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestIssueRefund_WithReason(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		var reason string
		json.Unmarshal(reqBody["reason"], &reason)
		if reason != "Customer requested" {
			t.Errorf("reason = %q, want %q", reason, "Customer requested")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"refund": map[string]any{"id": "REF789"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.issue_refund",
		Parameters:  json.RawMessage(`{"payment_id": "PMT123", "reason": "Customer requested"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestIssueRefund_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.issue_refund"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing payment_id", params: `{"amount_money": {"amount": 500, "currency": "USD"}}`},
		{name: "zero amount", params: `{"payment_id": "PMT123", "amount_money": {"amount": 0, "currency": "USD"}}`},
		{name: "negative amount", params: `{"payment_id": "PMT123", "amount_money": {"amount": -100, "currency": "USD"}}`},
		{name: "missing currency", params: `{"payment_id": "PMT123", "amount_money": {"amount": 500}}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.issue_refund",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestIssueRefund_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "INVALID_REQUEST_ERROR", "code": "REFUND_ALREADY_PENDING", "detail": "A refund is already pending for this payment"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.issue_refund"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.issue_refund",
		Parameters:  json.RawMessage(`{"payment_id": "PMT123"}`),
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

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.issue_refund"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "square.issue_refund",
		Parameters:  json.RawMessage(`{"payment_id": "PMT123"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
