package shopify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetOrder_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Path; got != "/orders/1001.json" {
			t.Errorf("path = %s, want /orders/1001.json", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"order": map[string]any{
				"id": 1001, "name": "#1001", "total_price": "99.99",
				"line_items": []map[string]any{
					{"id": 1, "title": "Widget", "quantity": 2},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_order"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_order",
		Parameters:  json.RawMessage(`{"order_id":1001}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["order"]; !ok {
		t.Error("result missing 'order' key")
	}
}

func TestGetOrder_InvalidOrderID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.get_order"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing order_id", `{}`},
		{"zero order_id", `{"order_id":0}`},
		{"negative order_id", `{"order_id":-1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "shopify.get_order",
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

func TestGetOrder_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":"Not Found"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_order",
		Parameters:  json.RawMessage(`{"order_id":999999}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}

func TestGetOrder_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.get_order"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_order",
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
