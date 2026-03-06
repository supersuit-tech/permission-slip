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

func TestCreatePayment_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/payments" {
			t.Errorf("path = %s, want /payments", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		// Verify idempotency key is present (critical for payments).
		if _, ok := reqBody["idempotency_key"]; !ok {
			t.Error("missing idempotency_key in request body — required for payments to prevent double-charges")
		}

		var sourceID string
		json.Unmarshal(reqBody["source_id"], &sourceID)
		if sourceID != "cnon:card-nonce-ok" {
			t.Errorf("source_id = %q, want %q", sourceID, "cnon:card-nonce-ok")
		}

		var amountMoney money
		json.Unmarshal(reqBody["amount_money"], &amountMoney)
		if amountMoney.Amount != 1000 {
			t.Errorf("amount = %d, want 1000", amountMoney.Amount)
		}
		if amountMoney.Currency != "USD" {
			t.Errorf("currency = %q, want USD", amountMoney.Currency)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"payment": map[string]any{
				"id":          "PMT123",
				"status":      "COMPLETED",
				"total_money": map[string]any{"amount": 1000, "currency": "USD"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_payment"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.create_payment",
		Parameters:  json.RawMessage(`{"source_id": "cnon:card-nonce-ok", "amount_money": {"amount": 1000, "currency": "USD"}}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "PMT123" {
		t.Errorf("payment id = %v, want PMT123", data["id"])
	}
	if data["status"] != "COMPLETED" {
		t.Errorf("payment status = %v, want COMPLETED", data["status"])
	}
}

func TestCreatePayment_WithOptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		var orderID string
		json.Unmarshal(reqBody["order_id"], &orderID)
		if orderID != "ORD123" {
			t.Errorf("order_id = %q, want %q", orderID, "ORD123")
		}

		var customerID string
		json.Unmarshal(reqBody["customer_id"], &customerID)
		if customerID != "CUST123" {
			t.Errorf("customer_id = %q, want %q", customerID, "CUST123")
		}

		var note string
		json.Unmarshal(reqBody["note"], &note)
		if note != "Payment for order" {
			t.Errorf("note = %q, want %q", note, "Payment for order")
		}

		var refID string
		json.Unmarshal(reqBody["reference_id"], &refID)
		if refID != "REF-001" {
			t.Errorf("reference_id = %q, want %q", refID, "REF-001")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"payment": map[string]any{"id": "PMT456"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_payment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.create_payment",
		Parameters: json.RawMessage(`{
			"source_id": "CASH",
			"amount_money": {"amount": 2000, "currency": "USD"},
			"order_id": "ORD123",
			"customer_id": "CUST123",
			"note": "Payment for order",
			"reference_id": "REF-001"
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreatePayment_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.create_payment"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing source_id", params: `{"amount_money": {"amount": 1000, "currency": "USD"}}`},
		{name: "missing amount_money", params: `{"source_id": "CASH"}`},
		{name: "missing currency", params: `{"source_id": "CASH", "amount_money": {"amount": 1000}}`},
		{name: "zero amount", params: `{"source_id": "CASH", "amount_money": {"amount": 0, "currency": "USD"}}`},
		{name: "negative amount", params: `{"source_id": "CASH", "amount_money": {"amount": -500, "currency": "USD"}}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.create_payment",
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

func TestCreatePayment_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "INVALID_REQUEST_ERROR", "code": "INVALID_CARD_DATA", "detail": "Invalid card nonce"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_payment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.create_payment",
		Parameters:  json.RawMessage(`{"source_id": "bad-nonce", "amount_money": {"amount": 1000, "currency": "USD"}}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreatePayment_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_payment"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "square.create_payment",
		Parameters:  json.RawMessage(`{"source_id": "CASH", "amount_money": {"amount": 1000, "currency": "USD"}}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
