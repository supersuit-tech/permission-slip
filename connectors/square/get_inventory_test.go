package square

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetInventory_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/inventory/counts/batch-retrieve" {
			t.Errorf("path = %s, want /inventory/counts/batch-retrieve", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		var ids []string
		json.Unmarshal(reqBody["catalog_object_ids"], &ids)
		if len(ids) != 2 || ids[0] != "ITEM1" || ids[1] != "ITEM2" {
			t.Errorf("catalog_object_ids = %v, want [ITEM1, ITEM2]", ids)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"counts": []map[string]any{
				{"catalog_object_id": "ITEM1", "quantity": "10", "state": "IN_STOCK"},
				{"catalog_object_id": "ITEM2", "quantity": "5", "state": "IN_STOCK"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.get_inventory"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.get_inventory",
		Parameters:  json.RawMessage(`{"catalog_object_ids": ["ITEM1", "ITEM2"]}`),
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
	if len(counts) != 2 {
		t.Errorf("counts length = %d, want 2", len(counts))
	}
}

func TestGetInventory_WithLocationFilter(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]json.RawMessage
		json.Unmarshal(body, &reqBody)

		var locationIDs []string
		json.Unmarshal(reqBody["location_ids"], &locationIDs)
		if len(locationIDs) != 1 || locationIDs[0] != "LOC1" {
			t.Errorf("location_ids = %v, want [LOC1]", locationIDs)
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
	action := conn.Actions()["square.get_inventory"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.get_inventory",
		Parameters:  json.RawMessage(`{"catalog_object_ids": ["ITEM1"], "location_ids": ["LOC1"]}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetInventory_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["square.get_inventory"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "square.get_inventory",
		Parameters:  json.RawMessage(`{"catalog_object_ids": ["ITEM1"]}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	counts, ok := data["counts"].([]any)
	if !ok {
		t.Fatalf("counts should be an array, got %T", data["counts"])
	}
	if len(counts) != 0 {
		t.Errorf("counts length = %d, want 0", len(counts))
	}
}

func TestGetInventory_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["square.get_inventory"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing catalog_object_ids", params: `{}`},
		{name: "empty catalog_object_ids", params: `{"catalog_object_ids": []}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "square.get_inventory",
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
