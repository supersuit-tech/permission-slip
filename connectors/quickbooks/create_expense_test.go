package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateExpense_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/company/1234567890/purchase" {
			t.Errorf("path = %s, want /v3/company/1234567890/purchase", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["PaymentType"] != "CreditCard" {
			t.Errorf("PaymentType = %v, want CreditCard", body["PaymentType"])
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Purchase": map[string]any{
				"Id":          "3001",
				"PaymentType": "CreditCard",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.create_expense"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "quickbooks.create_expense",
		Parameters: json.RawMessage(`{
			"account_id": "35",
			"payment_type": "CreditCard",
			"lines": [{"description": "Office Supplies", "amount": 75.50, "account_id": "20"}]
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
	if data["Id"] != "3001" {
		t.Errorf("Id = %v, want 3001", data["Id"])
	}
}

func TestCreateExpense_DefaultPaymentType(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["PaymentType"] != "Cash" {
			t.Errorf("PaymentType = %v, want Cash (default)", body["PaymentType"])
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Purchase": map[string]any{"Id": "3002"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.create_expense"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_expense",
		Parameters:  json.RawMessage(`{"account_id":"35","lines":[{"description":"Lunch","amount":25,"account_id":"20"}]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateExpense_MissingAccountID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_expense"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_expense",
		Parameters:  json.RawMessage(`{"lines":[{"description":"Test","amount":50,"account_id":"20"}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateExpense_EmptyLines(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_expense"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_expense",
		Parameters:  json.RawMessage(`{"account_id":"35","lines":[]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateExpense_InvalidPaymentType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_expense"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_expense",
		Parameters:  json.RawMessage(`{"account_id":"35","payment_type":"Bitcoin","lines":[{"description":"Test","amount":50,"account_id":"20"}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid payment_type, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
