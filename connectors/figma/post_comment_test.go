package figma

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestPostComment_Success(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/files/abc123/comments" {
			t.Errorf("expected path /files/abc123/comments, got %s", r.URL.Path)
		}

		var body postCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Message != "Great design!" {
			t.Errorf("expected message 'Great design!', got %q", body.Message)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":      "comment-new",
			"message": body.Message,
		})
	})

	action := &postCommentAction{conn: conn}
	params, _ := json.Marshal(postCommentParams{
		FileKey: "abc123",
		Message: "Great design!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.post_comment",
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
	if data["id"] != "comment-new" {
		t.Errorf("expected id 'comment-new', got %v", data["id"])
	}
}

func TestPostComment_WithReply(t *testing.T) {
	t.Parallel()

	_, conn := newTestServer(t, func(w http.ResponseWriter, r *http.Request) {
		var body postCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.CommentID != "parent-123" {
			t.Errorf("expected comment_id 'parent-123', got %q", body.CommentID)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"id": "reply-1"})
	})

	action := &postCommentAction{conn: conn}
	params, _ := json.Marshal(postCommentParams{
		FileKey:   "abc123",
		Message:   "I agree!",
		CommentID: "parent-123",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.post_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPostComment_MissingMessage(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &postCommentAction{conn: conn}
	params, _ := json.Marshal(map[string]any{"file_key": "abc123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.post_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing message")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestPostComment_MissingFileKey(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &postCommentAction{conn: conn}
	params, _ := json.Marshal(map[string]any{"message": "hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.post_comment",
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
