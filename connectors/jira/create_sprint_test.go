package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateSprint_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/sprint" {
			t.Errorf("path = %s, want /sprint", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		if reqBody["name"] != "Sprint 1" {
			t.Errorf("name = %v, want Sprint 1", reqBody["name"])
		}
		if reqBody["originBoardId"] != float64(42) {
			t.Errorf("originBoardId = %v, want 42", reqBody["originBoardId"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    1,
			"name":  "Sprint 1",
			"state": "future",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.create_sprint"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.create_sprint",
		Parameters:  json.RawMessage(`{"name":"Sprint 1","board_id":42}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	json.Unmarshal(result.Data, &data)
	if data["name"] != "Sprint 1" {
		t.Errorf("name = %v, want Sprint 1", data["name"])
	}
}

func TestCreateSprint_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.create_sprint"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.create_sprint",
		Parameters:  json.RawMessage(`{"board_id":42}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateSprint_MissingBoardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.create_sprint"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.create_sprint",
		Parameters:  json.RawMessage(`{"name":"Sprint 1"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing board_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
