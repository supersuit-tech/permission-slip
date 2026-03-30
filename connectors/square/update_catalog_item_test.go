package square

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateCatalogItem_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/catalog/object" {
			t.Errorf("path = %s, want /catalog/object", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		if _, ok := reqBody["idempotency_key"]; !ok {
			t.Error("missing idempotency_key in request body")
		}

		var obj map[string]json.RawMessage
		json.Unmarshal(reqBody["object"], &obj)

		var objType string
		json.Unmarshal(obj["type"], &objType)
		if objType != "ITEM" {
			t.Errorf("object type = %q, want ITEM", objType)
		}

		var objID string
		json.Unmarshal(obj["id"], &objID)
		if objID != "CATALOG123" {
			t.Errorf("object id = %q, want CATALOG123", objID)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"catalog_object": map[string]any{
				"type":    "ITEM",
				"id":      "CATALOG123",
				"version": 2,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.update_catalog_item"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.update_catalog_item",
		Parameters:  json.RawMessage(`{"object_id": "CATALOG123", "name": "Updated Item", "description": "New description"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "CATALOG123" {
		t.Errorf("catalog object id = %v, want CATALOG123", data["id"])
	}
}

func TestUpdateCatalogItem_WithVariations(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		var obj map[string]json.RawMessage
		json.Unmarshal(reqBody["object"], &obj)

		var itemData map[string]json.RawMessage
		json.Unmarshal(obj["item_data"], &itemData)

		if _, ok := itemData["variations"]; !ok {
			t.Error("expected variations in item_data")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"catalog_object": map[string]any{
				"type": "ITEM",
				"id":   "CATALOG123",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.update_catalog_item"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.update_catalog_item",
		Parameters: json.RawMessage(`{
			"object_id": "CATALOG123",
			"name": "Coffee",
			"variations": [
				{"id": "VAR1", "name": "Small", "pricing_type": "FIXED_PRICING", "price_money": {"amount": 350, "currency": "USD"}},
				{"id": "VAR2", "name": "Large", "pricing_type": "FIXED_PRICING", "price_money": {"amount": 500, "currency": "USD"}}
			]
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestUpdateCatalogItem_WithVersion(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		var obj map[string]json.RawMessage
		json.Unmarshal(reqBody["object"], &obj)

		var version float64
		json.Unmarshal(obj["version"], &version)
		if version != 5 {
			t.Errorf("version = %v, want 5", version)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"catalog_object": map[string]any{"id": "CATALOG123"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.update_catalog_item"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.update_catalog_item",
		Parameters:  json.RawMessage(`{"object_id": "CATALOG123", "name": "Updated", "version": 5}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestUpdateCatalogItem_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.update_catalog_item"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing object_id", params: `{"name": "Test"}`},
		{name: "no fields to update", params: `{"object_id": "X"}`},
		{name: "variation missing id", params: `{"object_id": "X", "variations": [{"name": "Small"}]}`},
		{name: "variation negative price", params: `{"object_id": "X", "variations": [{"id": "V1", "price_money": {"amount": -100, "currency": "USD"}}]}`},
		{name: "variation missing currency", params: `{"object_id": "X", "variations": [{"id": "V1", "price_money": {"amount": 100}}]}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.update_catalog_item",
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
