package figma

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetComponents_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/files/abc123/components" {
			t.Errorf("expected path /files/abc123/components, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"meta": map[string]any{
				"components": []map[string]any{
					{"key": "comp1", "name": "Button"},
				},
			},
		})
	})

	action := &getComponentsAction{conn: conn}
	params, _ := json.Marshal(getComponentsParams{FileKey: "abc123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_components",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["meta"] == nil {
		t.Error("expected meta in response")
	}
}

func TestGetComponents_MissingFileKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getComponentsAction{conn: conn}
	params, _ := json.Marshal(map[string]any{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.get_components",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing file_key")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
