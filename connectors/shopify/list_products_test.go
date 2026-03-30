package shopify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListProducts_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/products.json" {
			t.Errorf("path = %s, want /products.json", r.URL.Path)
		}
		// Default limit should be applied.
		if got := r.URL.Query().Get("limit"); got == "" {
			t.Error("expected limit query param")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"products": []map[string]any{
				{"id": 1, "title": "Widget"},
				{"id": 2, "title": "Gadget"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.list_products"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.list_products",
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
	if _, ok := data["products"]; !ok {
		t.Error("result missing 'products' key")
	}
}

func TestListProducts_WithFilters(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("status"); got != "active" {
			t.Errorf("status = %q, want %q", got, "active")
		}
		if got := r.URL.Query().Get("vendor"); got != "Acme" {
			t.Errorf("vendor = %q, want %q", got, "Acme")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"products": []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.list_products"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.list_products",
		Parameters:  json.RawMessage(`{"status": "active", "vendor": "Acme"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestListProducts_InvalidStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.list_products"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.list_products",
		Parameters:  json.RawMessage(`{"status": "invalid_status"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid status, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestListProducts_InvalidLimit(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.list_products"]

	tests := []struct {
		name   string
		params string
	}{
		{"limit too large", `{"limit": 300}`},
		{"negative limit", `{"limit": -1}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "shopify.list_products",
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
