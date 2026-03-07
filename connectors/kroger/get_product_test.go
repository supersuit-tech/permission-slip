package kroger

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
		if got := r.URL.Path; got != "/products/0001111041700" {
			t.Errorf("path = %s, want /products/0001111041700", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test_access_token_123" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer test_access_token_123")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"productId":   "0001111041700",
				"description": "Kroger 2% Milk",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.get_product"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.get_product",
		Parameters:  json.RawMessage(`{"product_id":"0001111041700"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	product, ok := data["data"].(map[string]any)
	if !ok {
		t.Fatal("expected data object in response")
	}
	if product["productId"] != "0001111041700" {
		t.Errorf("productId = %v, want 0001111041700", product["productId"])
	}
}

func TestGetProduct_WithLocation(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("filter.locationId"); got != "01400376" {
			t.Errorf("filter.locationId = %q, want %q", got, "01400376")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["kroger.get_product"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.get_product",
		Parameters:  json.RawMessage(`{"product_id":"0001111041700","location_id":"01400376"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetProduct_MissingProductID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.get_product"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.get_product",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestGetProduct_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["kroger.get_product"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "kroger.get_product",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
