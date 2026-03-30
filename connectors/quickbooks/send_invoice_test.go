package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendInvoice_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/company/1234567890/invoice/1001/send" {
			t.Errorf("path = %s, want /v3/company/1234567890/invoice/1001/send", r.URL.Path)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Invoice": map[string]any{
				"Id":            "1001",
				"EmailStatus":   "EmailSent",
				"DeliveryInfo":  map[string]any{"DeliveryType": "Email"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.send_invoice"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.send_invoice",
		Parameters:  json.RawMessage(`{"invoice_id": "1001"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["Id"] != "1001" {
		t.Errorf("Id = %v, want 1001", data["Id"])
	}
}

func TestSendInvoice_WithEmailTo(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "sendTo=") {
			t.Errorf("expected sendTo query param, got: %q", r.URL.RawQuery)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Invoice": map[string]any{"Id": "1001"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.send_invoice"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.send_invoice",
		Parameters:  json.RawMessage(`{"invoice_id": "1001", "email_to": "customer@example.com"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSendInvoice_MissingInvoiceID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.send_invoice"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing invoice_id", `{}`},
		{"empty invoice_id", `{"invoice_id": ""}`},
		{"invalid JSON", `{bad}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "quickbooks.send_invoice",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestSendInvoice_PathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.send_invoice"]

	malicious := []string{
		"1001/../../admin",
		"../secret",
		"1001'; DROP TABLE invoices--",
	}
	for _, id := range malicious {
		_, err := action.Execute(t.Context(), connectors.ActionRequest{
			ActionType:  "quickbooks.send_invoice",
			Parameters:  []byte(`{"invoice_id": "` + id + `"}`),
			Credentials: validCreds(),
		})
		if err == nil {
			t.Errorf("invoice_id %q: expected error, got nil", id)
		}
		if !connectors.IsValidationError(err) {
			t.Errorf("invoice_id %q: expected ValidationError, got %T: %v", id, err, err)
		}
	}
}

func TestSendInvoice_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"Fault": map[string]any{
				"Error": []map[string]any{
					{"Message": "Object Not Found", "Detail": "Invoice 9999 not found", "code": "610"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.send_invoice"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.send_invoice",
		Parameters:  json.RawMessage(`{"invoice_id": "9999"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}

func TestSendInvoice_InvalidEmailTo(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.send_invoice"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.send_invoice",
		Parameters:  json.RawMessage(`{"invoice_id": "1001", "email_to": "not-an-email"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid email_to, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
