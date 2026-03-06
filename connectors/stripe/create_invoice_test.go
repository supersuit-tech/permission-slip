package stripe

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateInvoice_Success_WithLineItemsAndFinalize(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := callCount.Add(1)

		switch {
		case call == 1 && r.URL.Path == "/v1/invoices":
			if r.Method != http.MethodPost {
				t.Errorf("method = %s, want POST", r.Method)
			}
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm: %v", err)
			}
			if got := r.FormValue("customer"); got != "cus_123" {
				t.Errorf("customer = %q, want %q", got, "cus_123")
			}
			if got := r.FormValue("description"); got != "Monthly invoice" {
				t.Errorf("description = %q, want %q", got, "Monthly invoice")
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"id":     "inv_abc",
				"status": "draft",
			})

		case call == 2 && r.URL.Path == "/v1/invoiceitems":
			if err := r.ParseForm(); err != nil {
				t.Fatalf("ParseForm: %v", err)
			}
			if got := r.FormValue("invoice"); got != "inv_abc" {
				t.Errorf("invoice = %q, want %q", got, "inv_abc")
			}
			if got := r.FormValue("amount"); got != "1000" {
				t.Errorf("amount = %q, want %q", got, "1000")
			}
			if got := r.FormValue("description"); got != "Widget" {
				t.Errorf("description = %q, want %q", got, "Widget")
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"id": "ii_1"})

		case call == 3 && r.URL.Path == "/v1/invoices/inv_abc/finalize":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"id":     "inv_abc",
				"status": "open",
			})

		default:
			t.Errorf("unexpected call %d: %s %s", call, r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_invoice"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "stripe.create_invoice",
		Parameters: json.RawMessage(`{
			"customer_id": "cus_123",
			"description": "Monthly invoice",
			"line_items": [{"description": "Widget", "amount": 1000}]
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
	if data["id"] != "inv_abc" {
		t.Errorf("id = %v, want inv_abc", data["id"])
	}
	if data["status"] != "open" {
		t.Errorf("status = %v, want open", data["status"])
	}

	if got := callCount.Load(); got != 3 {
		t.Errorf("callCount = %d, want 3 (create invoice + add item + finalize)", got)
	}
}

func TestCreateInvoice_NoFinalize(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := callCount.Add(1)

		switch {
		case call == 1 && r.URL.Path == "/v1/invoices":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"id":     "inv_draft",
				"status": "draft",
			})
		default:
			t.Errorf("unexpected call %d: %s %s", call, r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_invoice"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_invoice",
		Parameters:  json.RawMessage(`{"customer_id":"cus_123","auto_advance":false}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "draft" {
		t.Errorf("status = %v, want draft", data["status"])
	}
	if got := callCount.Load(); got != 1 {
		t.Errorf("callCount = %d, want 1 (create invoice only)", got)
	}
}

func TestCreateInvoice_MissingCustomerID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_invoice",
		Parameters:  json.RawMessage(`{"description":"test"}`),
		Credentials: validCreds(),
	})
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateInvoice_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["stripe.create_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "stripe.create_invoice",
		Parameters:  json.RawMessage(`{bad`),
		Credentials: validCreds(),
	})
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateInvoice_LineItemError(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		call := callCount.Add(1)
		switch {
		case call == 1 && r.URL.Path == "/v1/invoices":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"id":     "inv_fail",
				"status": "draft",
			})
		case call == 2 && r.URL.Path == "/v1/invoiceitems":
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{
					"type":    "invalid_request_error",
					"message": "Amount must be positive",
				},
			})
		default:
			t.Errorf("unexpected call %d", call)
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["stripe.create_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "stripe.create_invoice",
		Parameters: json.RawMessage(`{
			"customer_id": "cus_123",
			"line_items": [{"description": "Bad item", "amount": -100}]
		}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
}
