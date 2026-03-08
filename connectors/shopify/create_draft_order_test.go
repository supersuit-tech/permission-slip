package shopify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateDraftOrder_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/draft_orders.json" {
			t.Errorf("path = %s, want /draft_orders.json", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		draftOrder, ok := body["draft_order"].(map[string]any)
		if !ok {
			t.Error("missing 'draft_order' in request body")
		}
		lineItems, ok := draftOrder["line_items"].([]any)
		if !ok || len(lineItems) != 1 {
			t.Errorf("expected 1 line item, got %v", draftOrder["line_items"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"draft_order": map[string]any{
				"id":     10001,
				"status": "open",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_draft_order"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_draft_order",
		Parameters:  json.RawMessage(`{"line_items": [{"variant_id": 123, "quantity": 2}]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["draft_order"]; !ok {
		t.Error("result missing 'draft_order' key")
	}
}

func TestCreateDraftOrder_ValidationErrors(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_draft_order"]

	tests := []struct {
		name   string
		params string
	}{
		{"no line items", `{}`},
		{"empty line items", `{"line_items": []}`},
		{"zero quantity", `{"line_items": [{"variant_id": 1, "quantity": 0}]}`},
		{"negative quantity", `{"line_items": [{"variant_id": 1, "quantity": -1}]}`},
		{"missing identifier", `{"line_items": [{"quantity": 1}]}`},
		{"invalid JSON", `{bad}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "shopify.create_draft_order",
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

func TestCreateDraftOrder_WithCustomer(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		draftOrder := body["draft_order"].(map[string]any)
		customer, ok := draftOrder["customer"].(map[string]any)
		if !ok {
			t.Error("missing 'customer' in draft_order")
		}
		if customer["id"] != float64(9001) {
			t.Errorf("customer id = %v, want 9001", customer["id"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"draft_order": map[string]any{"id": 10002},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_draft_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "shopify.create_draft_order",
		Parameters: json.RawMessage(`{
			"line_items": [{"title": "Custom Item", "quantity": 1, "price": "29.99"}],
			"customer_id": 9001,
			"note": "Rush order"
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestCreateDraftOrder_HTTPError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte(`{"errors":{"line_items":["variant_id is not valid"]}}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.create_draft_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_draft_order",
		Parameters:  json.RawMessage(`{"line_items": [{"variant_id": 999, "quantity": 1}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateDraftOrder_CustomItemRequiresPrice(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.create_draft_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.create_draft_order",
		Parameters:  json.RawMessage(`{"line_items": [{"title": "Custom Widget", "quantity": 1}]}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for custom item without price, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
