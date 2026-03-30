package linear

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestAddComment_Success(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"commentCreate": map[string]any{
					"success": true,
					"comment": map[string]string{
						"id":   "comment-uuid-1",
						"body": "This is a comment",
						"url":  "https://linear.app/team/issue/ENG-123#comment-uuid-1",
					},
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &addCommentAction{conn: conn}

	params, _ := json.Marshal(addCommentParams{
		IssueID: "issue-uuid-1",
		Body:    "This is a comment",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.add_comment",
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
	if data["id"] != "comment-uuid-1" {
		t.Errorf("id = %q, want %q", data["id"], "comment-uuid-1")
	}
	if data["body"] != "This is a comment" {
		t.Errorf("body = %q, want %q", data["body"], "This is a comment")
	}
}

func TestAddComment_MissingIssueID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addCommentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"body": "A comment"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.add_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing issue_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddComment_MissingBody(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addCommentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"issue_id": "issue-1"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.add_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing body")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestAddComment_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &addCommentAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.add_comment",
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

func TestAddComment_SuccessFalse(t *testing.T) {
	t.Parallel()

	handler := &graphQLHandler{
		t: t,
		response: map[string]any{
			"data": map[string]any{
				"commentCreate": map[string]any{
					"success": false,
					"comment": nil,
				},
			},
		},
	}

	conn, _ := newTestServer(t, handler)
	action := &addCommentAction{conn: conn}

	params, _ := json.Marshal(addCommentParams{
		IssueID: "issue-1",
		Body:    "A comment",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linear.add_comment",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for success=false")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}
