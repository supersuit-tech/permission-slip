package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestReplyInstagramComment_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/17858893000000001/replies" {
			t.Errorf("expected path /17858893000000001/replies, got %s", r.URL.Path)
		}

		var body replyInstagramCommentRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if body.Message != "Thanks for commenting!" {
			t.Errorf("expected message 'Thanks for commenting!', got %q", body.Message)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(replyInstagramCommentResponse{ID: "17858893000000002"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &replyInstagramCommentAction{conn: conn}

	params, _ := json.Marshal(replyInstagramCommentParams{
		CommentID: "17858893000000001",
		Message:   "Thanks for commenting!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.reply_instagram_comment",
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
	if data["id"] != "17858893000000002" {
		t.Errorf("expected id '17858893000000002', got %q", data["id"])
	}
}

func TestReplyInstagramComment_MissingCommentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &replyInstagramCommentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"message": "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.reply_instagram_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestReplyInstagramComment_MissingMessage(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &replyInstagramCommentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"comment_id": "17858893000000001"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.reply_instagram_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
