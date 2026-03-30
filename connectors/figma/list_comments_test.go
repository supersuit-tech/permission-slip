package figma

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListComments_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/files/abc123/comments" {
			t.Errorf("expected path /files/abc123/comments, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"comments": []map[string]any{
				{"id": "comment-1", "message": "Looks good!"},
			},
		})
	})

	action := &listCommentsAction{conn: conn}
	params, _ := json.Marshal(listCommentsParams{FileKey: "abc123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.list_comments",
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
	comments, ok := data["comments"].([]any)
	if !ok {
		t.Fatal("expected comments array in response")
	}
	if len(comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(comments))
	}
}

func TestListComments_WithMarkdown(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("as_md"); got != "true" {
			t.Errorf("expected as_md=true, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"comments": []any{}})
	})

	action := &listCommentsAction{conn: conn}
	params, _ := json.Marshal(listCommentsParams{FileKey: "abc123", AsMd: true})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.list_comments",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListComments_MissingFileKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listCommentsAction{conn: conn}
	params, _ := json.Marshal(map[string]any{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.list_comments",
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
