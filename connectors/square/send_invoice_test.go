package square

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSendInvoice_Success(t *testing.T) {
	t.Parallel()

	var callCount atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		if _, ok := reqBody["idempotency_key"]; !ok {
			t.Error("missing idempotency_key in request body")
		}

		// Verify deterministic keys have correct step suffixes.
		var key string
		json.Unmarshal(reqBody["idempotency_key"], &key)

		callCount.Add(1)
		switch {
		case r.URL.Path == "/orders":
			if !strings.HasSuffix(key, "-order") {
				t.Errorf("order idempotency_key should end with -order, got %q", key)
			}
			// Step 1: Create order
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"order": map[string]any{
					"id":          "ORD123",
					"location_id": "LOC1",
				},
			})

		case r.URL.Path == "/invoices":
			if !strings.HasSuffix(key, "-invoice") {
				t.Errorf("invoice idempotency_key should end with -invoice, got %q", key)
			}
			// Step 2: Create invoice
			var invoice map[string]json.RawMessage
			json.Unmarshal(reqBody["invoice"], &invoice)

			var orderID string
			json.Unmarshal(invoice["order_id"], &orderID)
			if orderID != "ORD123" {
				t.Errorf("invoice order_id = %q, want ORD123", orderID)
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"invoice": map[string]any{
					"id":      "INV123",
					"version": 0,
					"status":  "DRAFT",
				},
			})

		case strings.HasPrefix(r.URL.Path, "/invoices/") && strings.HasSuffix(r.URL.Path, "/publish"):
			if !strings.HasSuffix(key, "-publish") {
				t.Errorf("publish idempotency_key should end with -publish, got %q", key)
			}
			// Step 3: Publish invoice
			expectedPath := "/invoices/INV123/publish"
			if r.URL.Path != expectedPath {
				t.Errorf("path = %s, want %s", r.URL.Path, expectedPath)
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"invoice": map[string]any{
					"id":      "INV123",
					"version": 1,
					"status":  "SENT",
				},
			})

		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.send_invoice"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.send_invoice",
		Parameters: json.RawMessage(`{
			"customer_id": "CUST1",
			"location_id": "LOC1",
			"line_items": [
				{"description": "Consulting", "quantity": "2", "base_price_money": {"amount": 5000, "currency": "USD"}}
			],
			"due_date": "2024-12-31"
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	if c := callCount.Load(); c != 3 {
		t.Errorf("expected 3 API calls (order + invoice + publish), got %d", c)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "INV123" {
		t.Errorf("invoice id = %v, want INV123", data["id"])
	}
	if data["status"] != "SENT" {
		t.Errorf("invoice status = %v, want SENT", data["status"])
	}
}

func TestSendInvoice_WithOptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		switch {
		case r.URL.Path == "/orders":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"order": map[string]any{"id": "ORD1"},
			})

		case r.URL.Path == "/invoices":
			var invoice map[string]json.RawMessage
			json.Unmarshal(reqBody["invoice"], &invoice)

			var deliveryMethod string
			json.Unmarshal(invoice["delivery_method"], &deliveryMethod)
			if deliveryMethod != "SMS" {
				t.Errorf("delivery_method = %q, want SMS", deliveryMethod)
			}

			var title string
			json.Unmarshal(invoice["title"], &title)
			if title != "March Invoice" {
				t.Errorf("title = %q, want %q", title, "March Invoice")
			}

			var description string
			json.Unmarshal(invoice["description"], &description)
			if description != "Thank you" {
				t.Errorf("description = %q, want %q", description, "Thank you")
			}

			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"invoice": map[string]any{"id": "INV1", "version": 0},
			})

		case strings.HasSuffix(r.URL.Path, "/publish"):
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"invoice": map[string]any{"id": "INV1", "status": "SENT"},
			})
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.send_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.send_invoice",
		Parameters: json.RawMessage(`{
			"customer_id": "CUST1",
			"location_id": "LOC1",
			"line_items": [{"description": "Work", "quantity": "1", "base_price_money": {"amount": 1000, "currency": "USD"}}],
			"due_date": "2024-12-31",
			"delivery_method": "SMS",
			"title": "March Invoice",
			"note": "Thank you"
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSendInvoice_OrderCreationFails(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "INVALID_REQUEST_ERROR", "code": "INVALID_VALUE", "detail": "Invalid location"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.send_invoice"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.send_invoice",
		Parameters: json.RawMessage(`{
			"customer_id": "CUST1",
			"location_id": "BAD_LOC",
			"line_items": [{"description": "Work", "quantity": "1", "base_price_money": {"amount": 1000, "currency": "USD"}}],
			"due_date": "2024-12-31"
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
}

func TestSendInvoice_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.send_invoice"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing customer_id", params: `{"location_id":"L","line_items":[{"description":"X","quantity":"1","base_price_money":{"amount":100,"currency":"USD"}}],"due_date":"2024-01-01"}`},
		{name: "missing location_id", params: `{"customer_id":"C","line_items":[{"description":"X","quantity":"1","base_price_money":{"amount":100,"currency":"USD"}}],"due_date":"2024-01-01"}`},
		{name: "missing line_items", params: `{"customer_id":"C","location_id":"L","due_date":"2024-01-01"}`},
		{name: "empty line_items", params: `{"customer_id":"C","location_id":"L","line_items":[],"due_date":"2024-01-01"}`},
		{name: "missing due_date", params: `{"customer_id":"C","location_id":"L","line_items":[{"description":"X","quantity":"1","base_price_money":{"amount":100,"currency":"USD"}}]}`},
		{name: "invalid due_date format", params: `{"customer_id":"C","location_id":"L","line_items":[{"description":"X","quantity":"1","base_price_money":{"amount":100,"currency":"USD"}}],"due_date":"next Friday"}`},
		{name: "line item missing description", params: `{"customer_id":"C","location_id":"L","line_items":[{"quantity":"1","base_price_money":{"amount":100,"currency":"USD"}}],"due_date":"2024-01-01"}`},
		{name: "line item missing quantity", params: `{"customer_id":"C","location_id":"L","line_items":[{"description":"X","base_price_money":{"amount":100,"currency":"USD"}}],"due_date":"2024-01-01"}`},
		{name: "line item non-numeric quantity", params: `{"customer_id":"C","location_id":"L","line_items":[{"description":"X","quantity":"abc","base_price_money":{"amount":100,"currency":"USD"}}],"due_date":"2024-01-01"}`},
		{name: "line item zero quantity", params: `{"customer_id":"C","location_id":"L","line_items":[{"description":"X","quantity":"0","base_price_money":{"amount":100,"currency":"USD"}}],"due_date":"2024-01-01"}`},
		{name: "line item zero amount", params: `{"customer_id":"C","location_id":"L","line_items":[{"description":"X","quantity":"1","base_price_money":{"amount":0,"currency":"USD"}}],"due_date":"2024-01-01"}`},
		{name: "line item missing currency", params: `{"customer_id":"C","location_id":"L","line_items":[{"description":"X","quantity":"1","base_price_money":{"amount":100}}],"due_date":"2024-01-01"}`},
		{name: "invalid delivery_method", params: `{"customer_id":"C","location_id":"L","line_items":[{"description":"X","quantity":"1","base_price_money":{"amount":100,"currency":"USD"}}],"due_date":"2024-01-01","delivery_method":"PIGEON"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.send_invoice",
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
