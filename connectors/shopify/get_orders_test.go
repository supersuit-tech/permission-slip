package shopify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetOrders_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if got := r.URL.Path; got != "/orders.json" {
			t.Errorf("path = %s, want /orders.json", got)
		}
		if got := r.URL.Query().Get("status"); got != "open" {
			t.Errorf("status = %q, want %q", got, "open")
		}
		if got := r.URL.Query().Get("limit"); got != "50" {
			t.Errorf("limit = %q, want %q", got, "50")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"orders": []map[string]any{
				{"id": 1001, "name": "#1001", "total_price": "99.99"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_orders"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_orders",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["orders"]; !ok {
		t.Error("result missing 'orders' key")
	}
}

func TestGetOrders_WithFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if got := q.Get("status"); got != "closed" {
			t.Errorf("status = %q, want %q", got, "closed")
		}
		if got := q.Get("financial_status"); got != "paid" {
			t.Errorf("financial_status = %q, want %q", got, "paid")
		}
		if got := q.Get("limit"); got != "10" {
			t.Errorf("limit = %q, want %q", got, "10")
		}
		if got := q.Get("created_at_min"); got != "2024-01-01T00:00:00Z" {
			t.Errorf("created_at_min = %q, want %q", got, "2024-01-01T00:00:00Z")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"orders": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_orders"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_orders",
		Parameters:  json.RawMessage(`{"status":"closed","financial_status":"paid","limit":10,"created_at_min":"2024-01-01T00:00:00Z"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestGetOrders_InvalidStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.get_orders"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_orders",
		Parameters:  json.RawMessage(`{"status":"invalid"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGetOrders_InvalidLimit(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.get_orders"]

	tests := []struct {
		name   string
		params string
	}{
		{"limit too high", `{"limit":300}`},
		{"limit negative", `{"limit":-1}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "shopify.get_orders",
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

func TestGetOrders_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.get_orders"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_orders",
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

func TestGetOrders_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"errors":"Internal Server Error"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_orders"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_orders",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
