package square

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAdjustInventory_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/inventory/changes/batch-create" {
			t.Errorf("path = %s, want /inventory/changes/batch-create", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		if _, ok := reqBody["idempotency_key"]; !ok {
			t.Error("missing idempotency_key in request body")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"counts": []map[string]any{
				{"catalog_object_id": "ITEM1", "quantity": "10", "state": "IN_STOCK"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.adjust_inventory"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.adjust_inventory",
		Parameters: json.RawMessage(`{
			"catalog_object_id": "ITEM1",
			"location_id": "LOC1",
			"quantity": "10",
			"from_state": "NONE",
			"to_state": "IN_STOCK"
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
	counts, ok := data["counts"].([]any)
	if !ok {
		t.Fatalf("counts is not an array: %T", data["counts"])
	}
	if len(counts) != 1 {
		t.Errorf("counts length = %d, want 1", len(counts))
	}
}

func TestAdjustInventory_SameState(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		// When from_state == to_state, should use PHYSICAL_COUNT type.
		var changes []map[string]json.RawMessage
		json.Unmarshal(reqBody["changes"], &changes)
		if len(changes) != 1 {
			t.Fatalf("changes length = %d, want 1", len(changes))
		}

		var changeType string
		json.Unmarshal(changes[0]["type"], &changeType)
		if changeType != "PHYSICAL_COUNT" {
			t.Errorf("change type = %q, want PHYSICAL_COUNT", changeType)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"counts": []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.adjust_inventory"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "square.adjust_inventory",
		Parameters: json.RawMessage(`{
			"catalog_object_id": "ITEM1",
			"location_id": "LOC1",
			"quantity": "5",
			"from_state": "IN_STOCK",
			"to_state": "IN_STOCK"
		}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestAdjustInventory_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.adjust_inventory"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing catalog_object_id", params: `{"location_id":"L","quantity":"1","from_state":"NONE","to_state":"IN_STOCK"}`},
		{name: "missing location_id", params: `{"catalog_object_id":"I","quantity":"1","from_state":"NONE","to_state":"IN_STOCK"}`},
		{name: "missing quantity", params: `{"catalog_object_id":"I","location_id":"L","from_state":"NONE","to_state":"IN_STOCK"}`},
		{name: "missing from_state", params: `{"catalog_object_id":"I","location_id":"L","quantity":"1","to_state":"IN_STOCK"}`},
		{name: "missing to_state", params: `{"catalog_object_id":"I","location_id":"L","quantity":"1","from_state":"NONE"}`},
		{name: "invalid from_state", params: `{"catalog_object_id":"I","location_id":"L","quantity":"1","from_state":"INVALID","to_state":"IN_STOCK"}`},
		{name: "invalid to_state", params: `{"catalog_object_id":"I","location_id":"L","quantity":"1","from_state":"NONE","to_state":"INVALID"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.adjust_inventory",
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
