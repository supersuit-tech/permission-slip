package notion

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAppendBlocks_SuccessWithChildren(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/v1/blocks/page-123/children" {
			t.Errorf("expected path /v1/blocks/page-123/children, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		children, ok := body["children"].([]any)
		if !ok {
			t.Fatal("expected children array in request body")
		}
		if len(children) != 1 {
			t.Errorf("expected 1 child, got %d", len(children))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"object":  "list",
			"results": []map[string]any{{"object": "block", "id": "block-456"}},
		})
	})

	action := &appendBlocksAction{conn: conn}
	children := `[{"object":"block","type":"heading_2","heading_2":{"rich_text":[{"text":{"content":"Section"}}]}}]`
	params, _ := json.Marshal(appendBlocksParams{
		PageID:   "page-123",
		Children: json.RawMessage(children),
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.append_blocks",
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
	if data["object"] != "list" {
		t.Errorf("expected object 'list', got %v", data["object"])
	}
}

func TestAppendBlocks_SuccessWithText(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		children, ok := body["children"].([]any)
		if !ok {
			t.Fatal("expected children array in request body")
		}
		if len(children) != 1 {
			t.Errorf("expected 1 auto-wrapped paragraph block, got %d", len(children))
		}
		block, ok := children[0].(map[string]any)
		if !ok {
			t.Fatal("expected block to be a map")
		}
		if block["type"] != "paragraph" {
			t.Errorf("expected type 'paragraph', got %v", block["type"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"object": "list", "results": []any{}})
	})

	action := &appendBlocksAction{conn: conn}
	params, _ := json.Marshal(appendBlocksParams{
		PageID: "page-123",
		Text:   "Hello, this is a log entry.",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.append_blocks",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestAppendBlocks_MissingPageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &appendBlocksAction{conn: conn}
	params, _ := json.Marshal(map[string]string{"text": "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.append_blocks",
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

func TestAppendBlocks_NoContent(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &appendBlocksAction{conn: conn}
	params, _ := json.Marshal(map[string]string{"page_id": "page-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.append_blocks",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error when no content provided")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAppendBlocks_MalformedChildren(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &appendBlocksAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"page_id":  "page-123",
		"children": "not-an-array",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.append_blocks",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for malformed children")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAppendBlocks_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &appendBlocksAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.append_blocks",
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
