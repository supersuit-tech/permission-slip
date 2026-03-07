package shopify

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateInventory_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/inventory_levels/adjust.json" {
			t.Errorf("path = %s, want /inventory_levels/adjust.json", got)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal request body: %v", err)
		}
		if reqBody["inventory_item_id"] != float64(808950810) {
			t.Errorf("inventory_item_id = %v, want 808950810", reqBody["inventory_item_id"])
		}
		if reqBody["location_id"] != float64(905684977) {
			t.Errorf("location_id = %v, want 905684977", reqBody["location_id"])
		}
		if reqBody["available_adjustment"] != float64(-5) {
			t.Errorf("available_adjustment = %v, want -5", reqBody["available_adjustment"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"inventory_level": map[string]any{
				"inventory_item_id": 808950810,
				"location_id":      905684977,
				"available":        95,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.update_inventory"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_inventory",
		Parameters:  json.RawMessage(`{"inventory_item_id":808950810,"location_id":905684977,"available_adjustment":-5}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	var data map[string]json.RawMessage
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if _, ok := data["inventory_level"]; !ok {
		t.Error("result missing 'inventory_level' key")
	}
}

func TestUpdateInventory_PositiveAdjustment(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if reqBody["available_adjustment"] != float64(10) {
			t.Errorf("available_adjustment = %v, want 10", reqBody["available_adjustment"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"inventory_level": map[string]any{"available": 110},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.update_inventory"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_inventory",
		Parameters:  json.RawMessage(`{"inventory_item_id":1,"location_id":2,"available_adjustment":10}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
}

func TestUpdateInventory_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.update_inventory"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing inventory_item_id", `{"location_id":1,"available_adjustment":5}`},
		{"missing location_id", `{"inventory_item_id":1,"available_adjustment":5}`},
		{"zero adjustment", `{"inventory_item_id":1,"location_id":1,"available_adjustment":0}`},
		{"all missing", `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "shopify.update_inventory",
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

func TestUpdateInventory_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["shopify.update_inventory"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_inventory",
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

func TestUpdateInventory_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"errors":"Not Found"}`))
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["shopify.update_inventory"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "shopify.update_inventory",
		Parameters:  json.RawMessage(`{"inventory_item_id":999,"location_id":999,"available_adjustment":1}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T: %v", err, err)
	}
}
