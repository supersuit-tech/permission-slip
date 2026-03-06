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

func TestCreateOrder_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/orders" {
			t.Errorf("path = %s, want /orders", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		// Verify idempotency key is present.
		if _, ok := reqBody["idempotency_key"]; !ok {
			t.Error("missing idempotency_key in request body")
		}

		// Verify order structure.
		var order map[string]json.RawMessage
		if err := json.Unmarshal(reqBody["order"], &order); err != nil {
			t.Fatalf("unmarshaling order: %v", err)
		}

		var locationID string
		json.Unmarshal(order["location_id"], &locationID)
		if locationID != "L123" {
			t.Errorf("location_id = %q, want %q", locationID, "L123")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"order": map[string]any{
				"id":          "ORD123",
				"state":       "OPEN",
				"total_money": map[string]any{"amount": 500, "currency": "USD"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_order"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.create_order",
		Parameters: json.RawMessage(`{
			"location_id": "L123",
			"line_items": [{"name": "Latte", "quantity": "1", "base_price_money": {"amount": 500, "currency": "USD"}}]
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
	if data["id"] != "ORD123" {
		t.Errorf("order id = %v, want ORD123", data["id"])
	}
	if data["state"] != "OPEN" {
		t.Errorf("order state = %v, want OPEN", data["state"])
	}
}

func TestCreateOrder_WithOptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		var order map[string]json.RawMessage
		json.Unmarshal(reqBody["order"], &order)

		var customerID string
		json.Unmarshal(order["customer_id"], &customerID)
		if customerID != "CUST123" {
			t.Errorf("customer_id = %q, want %q", customerID, "CUST123")
		}

		var note string
		json.Unmarshal(order["note"], &note)
		if note != "Rush order" {
			t.Errorf("note = %q, want %q", note, "Rush order")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"order": map[string]any{"id": "ORD456"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_order"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.create_order",
		Parameters: json.RawMessage(`{
			"location_id": "L123",
			"line_items": [{"name": "Latte", "quantity": "1", "base_price_money": {"amount": 500, "currency": "USD"}}],
			"customer_id": "CUST123",
			"note": "Rush order"
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateOrder_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.create_order"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing location_id", params: `{"line_items": [{"name": "X", "quantity": "1", "base_price_money": {"amount": 100, "currency": "USD"}}]}`},
		{name: "missing line_items", params: `{"location_id": "L123"}`},
		{name: "empty line_items", params: `{"location_id": "L123", "line_items": []}`},
		{name: "line_item missing name", params: `{"location_id": "L123", "line_items": [{"quantity": "1", "base_price_money": {"amount": 100, "currency": "USD"}}]}`},
		{name: "line_item missing quantity", params: `{"location_id": "L123", "line_items": [{"name": "X", "base_price_money": {"amount": 100, "currency": "USD"}}]}`},
		{name: "line_item missing currency", params: `{"location_id": "L123", "line_items": [{"name": "X", "quantity": "1", "base_price_money": {"amount": 100}}]}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.create_order",
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

func TestCreateOrder_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]any{
			"errors": []map[string]string{
				{"category": "API_ERROR", "code": "INTERNAL_SERVER_ERROR", "detail": "Something went wrong"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_order"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.create_order",
		Parameters: json.RawMessage(`{
			"location_id": "L123",
			"line_items": [{"name": "Latte", "quantity": "1", "base_price_money": {"amount": 500, "currency": "USD"}}]
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCreateOrder_Timeout(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 1*time.Millisecond)
	defer cancel()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.create_order"]

	_, err := action.Execute(ctx, connectors.ActionRequest{
		ActionType: "square.create_order",
		Parameters: json.RawMessage(`{
			"location_id": "L123",
			"line_items": [{"name": "Latte", "quantity": "1", "base_price_money": {"amount": 500, "currency": "USD"}}]
		}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsTimeoutError(err) {
		t.Errorf("expected TimeoutError, got %T: %v", err, err)
	}
}
