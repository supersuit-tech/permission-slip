package notion

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreatePage_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/pages" {
			t.Errorf("expected path /v1/pages, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ntn_test_token_123" {
			t.Errorf("bad auth header: %q", got)
		}
		if got := r.Header.Get("Notion-Version"); got != notionVersion {
			t.Errorf("bad Notion-Version header: %q", got)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		parent, ok := body["parent"].(map[string]any)
		if !ok {
			t.Fatal("missing parent in request body")
		}
		if parent["page_id"] != "parent-123" {
			t.Errorf("expected parent_id 'parent-123', got %v", parent["page_id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"object": "page",
			"id":     "new-page-456",
			"url":    "https://www.notion.so/new-page-456",
		})
	})

	action := &createPageAction{conn: conn}
	params, _ := json.Marshal(createPageParams{
		ParentID: "parent-123",
		Title:    "Test Page",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.create_page",
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
	if data["id"] != "new-page-456" {
		t.Errorf("expected id 'new-page-456', got %v", data["id"])
	}
}

func TestCreatePage_WithContent(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		children, ok := body["children"].([]any)
		if !ok {
			t.Fatal("expected children in request body")
		}
		if len(children) != 1 {
			t.Errorf("expected 1 child block, got %d", len(children))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"object": "page", "id": "page-789"})
	})

	action := &createPageAction{conn: conn}
	content := `[{"object":"block","type":"paragraph","paragraph":{"rich_text":[{"text":{"content":"Hello"}}]}}]`
	params, _ := json.Marshal(createPageParams{
		ParentID: "parent-123",
		Title:    "Test Page",
		Content:  json.RawMessage(content),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.create_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreatePage_DatabaseParentType(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		parent, ok := body["parent"].(map[string]any)
		if !ok {
			t.Fatal("missing parent in request body")
		}
		// When parent_type is "database_id", the parent key should be "database_id"
		if parent["database_id"] != "db-456" {
			t.Errorf("expected database_id 'db-456', got %v (parent=%v)", parent["database_id"], parent)
		}
		if parent["page_id"] != nil {
			t.Errorf("expected no page_id key, but found %v", parent["page_id"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"object": "page", "id": "entry-789"})
	})

	action := &createPageAction{conn: conn}
	params, _ := json.Marshal(createPageParams{
		ParentID:   "db-456",
		ParentType: "database_id",
		Title:      "New Entry",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.create_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreatePage_InvalidParentType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPageAction{conn: conn}
	params, _ := json.Marshal(map[string]string{
		"parent_id":   "parent-123",
		"parent_type": "workspace",
		"title":       "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.create_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid parent_type")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePage_MissingParentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPageAction{conn: conn}
	params, _ := json.Marshal(map[string]string{"title": "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.create_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing parent_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePage_MissingTitle(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPageAction{conn: conn}
	params, _ := json.Marshal(map[string]string{"parent_id": "parent-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.create_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePage_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPageAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.create_page",
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

func TestCreatePage_MalformedProperties(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPageAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"parent_id":  "parent-123",
		"title":      "Test",
		"properties": "not-an-object",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.create_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for malformed properties")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePage_MalformedContent(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPageAction{conn: conn}
	params, _ := json.Marshal(map[string]any{
		"parent_id": "parent-123",
		"title":     "Test",
		"content":   "not-an-array",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.create_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for malformed content")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePage_APIError(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		w.Write(notionErrorBody(400, "validation_error", "Title is not provided."))
	})

	action := &createPageAction{conn: conn}
	params, _ := json.Marshal(createPageParams{
		ParentID: "parent-123",
		Title:    "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "notion.create_page",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for API error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
