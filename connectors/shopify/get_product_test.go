package shopify

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetProduct_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/products/2001.json" {
			t.Errorf("path = %s, want /products/2001.json", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"product": map[string]any{
				"id":    2001,
				"title": "Super Widget",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_product"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_product",
		Parameters:  json.RawMessage(`{"product_id": 2001}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["product"]; !ok {
		t.Error("result missing 'product' key")
	}
}

func TestGetProduct_WithFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("fields"); got != "id,title" {
			t.Errorf("fields = %q, want %q", got, "id,title")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"product": map[string]any{"id": 2001, "title": "Widget"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_product",
		Parameters:  json.RawMessage(`{"product_id": 2001, "fields": "id,title"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestGetProduct_MissingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.get_product"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing product_id", `{}`},
		{"zero product_id", `{"product_id": 0}`},
		{"negative product_id", `{"product_id": -5}`},
		{"invalid JSON", `{bad}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "shopify.get_product",
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

func TestGetProduct_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":"Not Found"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.get_product"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.get_product",
		Parameters:  json.RawMessage(`{"product_id": 999999}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}
