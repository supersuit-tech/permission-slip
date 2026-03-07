package trello

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddComment_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/cards/card123/actions/comments" {
			t.Errorf("expected path /cards/card123/actions/comments, got %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]string
		json.Unmarshal(body, &reqBody)
		if reqBody["text"] != "This is a comment" {
			t.Errorf("expected text='This is a comment', got %q", reqBody["text"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":   "action123",
			"type": "commentCard",
			"data": map[string]string{"text": "This is a comment"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["trello.add_comment"]

	params, _ := json.Marshal(addCommentParams{
		CardID: "card123",
		Text:   "This is a comment",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.add_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	json.Unmarshal(result.Data, &data)
	if data["id"] != "action123" {
		t.Errorf("expected id=action123, got %q", data["id"])
	}
	if data["text"] != "This is a comment" {
		t.Errorf("expected text='This is a comment', got %q", data["text"])
	}
}

func TestAddComment_MissingCardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.add_comment"]

	params, _ := json.Marshal(map[string]string{"text": "comment"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.add_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing card_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddComment_MissingText(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.add_comment"]

	params, _ := json.Marshal(map[string]string{"card_id": "card123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.add_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing text")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddComment_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["trello.add_comment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "trello.add_comment",
		Parameters:  []byte(`{bad`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
