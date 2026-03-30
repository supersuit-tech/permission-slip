package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateBill_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/company/1234567890/bill" {
			t.Errorf("path = %s, want /v3/company/1234567890/bill", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		vendorRef, ok := body["VendorRef"].(map[string]any)
		if !ok || vendorRef["value"] != "50" {
			t.Errorf("VendorRef = %v", body["VendorRef"])
		}
		lines, ok := body["Line"].([]any)
		if !ok || len(lines) != 2 {
			t.Errorf("expected 2 line items, got %v", body["Line"])
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Bill": map[string]any{
				"Id":        "200",
				"TotalAmt":  300.0,
				"VendorRef": map[string]any{"value": "50"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.create_bill"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "quickbooks.create_bill",
		Parameters: json.RawMessage(`{
			"vendor_id": "50",
			"due_date": "2025-12-31",
			"line_items": [
				{"description": "Office Supplies", "amount": 150.00},
				{"description": "Equipment", "amount": 150.00, "account_id": "100"}
			]
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["Id"] != "200" {
		t.Errorf("Id = %v, want 200", data["Id"])
	}
}

func TestCreateBill_ValidationErrors(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_bill"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing vendor_id", `{"line_items": [{"amount": 100}]}`},
		{"empty vendor_id", `{"vendor_id": "", "line_items": [{"amount": 100}]}`},
		{"no line items", `{"vendor_id": "50"}`},
		{"empty line items", `{"vendor_id": "50", "line_items": []}`},
		{"zero amount", `{"vendor_id": "50", "line_items": [{"amount": 0}]}`},
		{"negative amount", `{"vendor_id": "50", "line_items": [{"amount": -10}]}`},
		{"invalid due_date", `{"vendor_id": "50", "due_date": "12/31/2025", "line_items": [{"amount": 100}]}`},
		{"invalid JSON", `{bad}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "quickbooks.create_bill",
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
