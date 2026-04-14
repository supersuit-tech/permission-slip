package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateIssue_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/repos/o/r/issues/7" {
			t.Errorf("path = %s, want /repos/o/r/issues/7", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling: %v", err)
		}
		if reqBody["title"] != "new title" {
			t.Errorf("title = %v, want new title", reqBody["title"])
		}
		if reqBody["state"] != "closed" {
			t.Errorf("state = %v, want closed", reqBody["state"])
		}
		if _, has := reqBody["body"]; has {
			t.Errorf("body should not be set when empty, got %v", reqBody["body"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"number": 7, "title": "new title", "state": "closed",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.update_issue"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.update_issue",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","issue_number":7,"title":"new title","state":"closed"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["state"] != "closed" {
		t.Errorf("state = %v, want closed", data["state"])
	}
}

func TestUpdateIssue_NoFieldsIsError(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.update_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.update_issue",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","issue_number":7}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error when no fields set, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdateIssue_InvalidState(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.update_issue"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.update_issue",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","issue_number":7,"state":"invalid"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for invalid state, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
