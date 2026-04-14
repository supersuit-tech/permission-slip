package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdatePR_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("method = %s, want PATCH", r.Method)
		}
		if r.URL.Path != "/repos/o/r/pulls/12" {
			t.Errorf("path = %s, want /repos/o/r/pulls/12", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]any
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling: %v", err)
		}
		if reqBody["title"] != "updated" {
			t.Errorf("title = %v, want updated", reqBody["title"])
		}
		if reqBody["base"] != "develop" {
			t.Errorf("base = %v, want develop", reqBody["base"])
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"number": 12, "title": "updated", "state": "open",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.update_pr"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.update_pr",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","pull_number":12,"title":"updated","base":"develop"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestUpdatePR_NoFieldsIsError(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.update_pr"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.update_pr",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","pull_number":12}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error when no fields set, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUpdatePR_InvalidState(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.update_pr"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.update_pr",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","pull_number":12,"state":"invalid"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for invalid state, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
