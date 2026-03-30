package notion

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdatePage_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/v1/pages/page-123" {
			t.Errorf("expected path /v1/pages/page-123, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		props, ok := body["properties"].(map[string]any)
		if !ok {
			t.Fatal("expected properties in request body")
		}
		if props["Status"] == nil {
			t.Error("expected Status property in body")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"object": "page",
			"id":     "page-123",
		})
	})

	action := &updatePageAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"page_id":    "page-123",
		"properties": map[string]any{"Status": map[string]any{"select": map[string]string{"name": "Done"}}},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.update_page",
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
	if data["id"] != "page-123" {
		t.Errorf("expected id 'page-123', got %v", data["id"])
	}
}

func TestUpdatePage_Archive(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body["archived"] != true {
			t.Errorf("expected archived=true, got %v", body["archived"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"object": "page", "id": "page-123", "archived": true})
	})

	action := &updatePageAction{conn: conn}
	archived := true
	params, _ := json.Marshal(updatePageParams{
		PageID:   "page-123",
		Archived: &archived,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.update_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdatePage_MissingPageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updatePageAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"properties": map[string]any{"Status": "Done"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.update_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing page_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdatePage_NoUpdates(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updatePageAction{conn: conn}
	params, _ := json.Marshal(map[string]string{"page_id": "page-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.update_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when no updates provided")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdatePage_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updatePageAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.update_page",
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
