package quickbooks

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateInvoice_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/company/1234567890/invoice" {
			t.Errorf("path = %s, want /v3/company/1234567890/invoice", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		customerRef, ok := body["CustomerRef"].(map[string]any)
		if !ok {
			t.Errorf("missing CustomerRef")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if customerRef["value"] != "42" {
			t.Errorf("CustomerRef.value = %v, want 42", customerRef["value"])
		}

		lines, ok := body["Line"].([]any)
		if !ok || len(lines) != 2 {
			t.Errorf("expected 2 lines, got %v", body["Line"])
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Invoice": map[string]any{
				"Id":      "1001",
				"Balance": 250.0,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.create_invoice"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "quickbooks.create_invoice",
		Parameters: json.RawMessage(`{
			"customer_id": "42",
			"due_date": "2025-12-31",
			"line_items": [
				{"description": "Consulting", "amount": 150.00, "quantity": 1},
				{"description": "Support", "amount": 100.00, "quantity": 1}
			]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["Id"] != "1001" {
		t.Errorf("Id = %v, want 1001", data["Id"])
	}
}

func TestCreateInvoice_MissingCustomerID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "quickbooks.create_invoice",
		Parameters: json.RawMessage(`{"line_items": [{"description": "Test", "amount": 100}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateInvoice_MissingLineItems(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_invoice",
		Parameters:  json.RawMessage(`{"customer_id": "42"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateInvoice_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_invoice",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateInvoice_NegativeAmount(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_invoice",
		Parameters:  json.RawMessage(`{"customer_id":"42","line_items":[{"description":"Bad","amount":-10}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateInvoice_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.create_invoice"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType:  "quickbooks.create_invoice",
		Parameters:  json.RawMessage(`{"customer_id":"42","line_items":[{"description":"Test","amount":100}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
