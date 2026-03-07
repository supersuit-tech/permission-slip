package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestRecordPayment_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/company/1234567890/payment" {
			t.Errorf("path = %s, want /v3/company/1234567890/payment", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["TotalAmt"] != float64(500) {
			t.Errorf("TotalAmt = %v, want 500", body["TotalAmt"])
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Payment": map[string]any{
				"Id":       "5001",
				"TotalAmt": 500.0,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.record_payment"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.record_payment",
		Parameters:  json.RawMessage(`{"customer_id":"42","amount":500}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["Id"] != "5001" {
		t.Errorf("Id = %v, want 5001", data["Id"])
	}
}

func TestRecordPayment_WithInvoiceID(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		lines, ok := body["Line"].([]any)
		if !ok || len(lines) != 1 {
			t.Fatalf("expected 1 line for invoice link, got %v", body["Line"])
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Payment": map[string]any{"Id": "5002"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.record_payment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.record_payment",
		Parameters:  json.RawMessage(`{"customer_id":"42","amount":100,"invoice_id":"1001"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestRecordPayment_MissingCustomerID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.record_payment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.record_payment",
		Parameters:  json.RawMessage(`{"amount":100}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestRecordPayment_ZeroAmount(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.record_payment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.record_payment",
		Parameters:  json.RawMessage(`{"customer_id":"42","amount":0}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
