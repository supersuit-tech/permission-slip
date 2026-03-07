package asana

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
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/tasks/67890/stories" {
			t.Errorf("path = %s, want /tasks/67890/stories", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var envelope map[string]any
		json.Unmarshal(body, &envelope)
		data := envelope["data"].(map[string]any)
		if data["text"] != "This is a comment" {
			t.Errorf("text = %v, want %q", data["text"], "This is a comment")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"gid":  "99999",
				"text": "This is a comment",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.add_comment"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.add_comment",
		Parameters:  json.RawMessage(`{"task_id":"67890","text":"This is a comment"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["gid"] != "99999" {
		t.Errorf("gid = %v, want %q", data["gid"], "99999")
	}
}

func TestAddComment_HTMLText(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var envelope map[string]any
		json.Unmarshal(body, &envelope)
		data := envelope["data"].(map[string]any)
		if data["html_text"] != "<b>Bold comment</b>" {
			t.Errorf("html_text = %v, want %q", data["html_text"], "<b>Bold comment</b>")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"gid": "1", "text": "Bold comment"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.add_comment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.add_comment",
		Parameters:  json.RawMessage(`{"task_id":"1","html_text":"<b>Bold comment</b>"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestAddComment_MissingTaskID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.add_comment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.add_comment",
		Parameters:  json.RawMessage(`{"text":"hello"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAddComment_MissingTextAndHTMLText(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.add_comment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.add_comment",
		Parameters:  json.RawMessage(`{"task_id":"1"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
