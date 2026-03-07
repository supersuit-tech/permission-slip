package monday

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateSubitem_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}

		var body graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"create_subitem": map[string]any{
					"id":   "55001",
					"name": "Sub Task",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createSubitemAction{conn: conn}

	params, _ := json.Marshal(createSubitemParams{
		ParentItemID: "12345",
		ItemName:     "Sub Task",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_subitem",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "55001" {
		t.Errorf("expected id '55001', got %q", data["id"])
	}
	if data["name"] != "Sub Task" {
		t.Errorf("expected name 'Sub Task', got %q", data["name"])
	}
}

func TestCreateSubitem_WithColumnValues(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body graphQLRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("failed to decode request body: %v", err)
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		cv, ok := body.Variables["column_values"]
		if !ok {
			t.Error("expected column_values in variables")
		}
		if _, ok := cv.(string); !ok {
			t.Errorf("expected column_values to be a string, got %T", cv)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"create_subitem": map[string]any{
					"id":   "55002",
					"name": "Sub with cols",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createSubitemAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"parent_item_id": "12345",
		"item_name":      "Sub with cols",
		"column_values": map[string]any{
			"status": map[string]string{"label": "Working on it"},
		},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_subitem",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "55002" {
		t.Errorf("expected id '55002', got %q", data["id"])
	}
}

func TestCreateSubitem_MissingParentItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSubitemAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"item_name": "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_subitem",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing parent_item_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateSubitem_NonNumericParentItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSubitemAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"parent_item_id": "abc",
		"item_name":      "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_subitem",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-numeric parent_item_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateSubitem_MissingItemName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSubitemAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"parent_item_id": "12345",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_subitem",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing item_name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateSubitem_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSubitemAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "monday.create_subitem",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
