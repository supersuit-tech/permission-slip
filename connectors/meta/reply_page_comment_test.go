package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestReplyPageComment_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/comment_123/comments" {
			t.Errorf("expected path /comment_123/comments, got %s", r.URL.Path)
		}

		var body replyPageCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Message != "Thanks for the feedback!" {
			t.Errorf("expected message 'Thanks for the feedback!', got %q", body.Message)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(replyPageCommentResponse{ID: "reply_456"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &replyPageCommentAction{conn: conn}

	params, _ := json.Marshal(replyPageCommentParams{
		CommentID: "comment_123",
		Message:   "Thanks for the feedback!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.reply_page_comment",
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
	if data["id"] != "reply_456" {
		t.Errorf("expected id 'reply_456', got %q", data["id"])
	}
}

func TestReplyPageComment_MissingCommentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &replyPageCommentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"message": "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.reply_page_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing comment_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestReplyPageComment_MissingMessage(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &replyPageCommentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"comment_id": "123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.reply_page_comment",
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
