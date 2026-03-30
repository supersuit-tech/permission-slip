package asana

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateTask_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/tasks" {
			t.Errorf("path = %s, want /tasks", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer 0/abc123test" {
			t.Errorf("Authorization = %q, want %q", got, "Bearer 0/abc123test")
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want %q", got, "application/json")
		}

		body, _ := io.ReadAll(r.Body)
		var envelope map[string]any
		if err := json.Unmarshal(body, &envelope); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}
		data, ok := envelope["data"].(map[string]any)
		if !ok {
			t.Fatal("missing data envelope")
		}
		if data["name"] != "Fix login bug" {
			t.Errorf("name = %v, want %q", data["name"], "Fix login bug")
		}
		projects, ok := data["projects"].([]any)
		if !ok || len(projects) != 1 || projects[0] != "12345" {
			t.Errorf("projects = %v, want [12345]", data["projects"])
		}
		if data["assignee"] != "me" {
			t.Errorf("assignee = %v, want %q", data["assignee"], "me")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"gid":           "67890",
				"name":          "Fix login bug",
				"permalink_url": "https://app.asana.com/0/12345/67890",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.create_task"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.create_task",
		Parameters:  json.RawMessage(`{"project_id":"12345","name":"Fix login bug","assignee":"me"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["gid"] != "67890" {
		t.Errorf("gid = %v, want %q", data["gid"], "67890")
	}
	if data["permalink_url"] != "https://app.asana.com/0/12345/67890" {
		t.Errorf("permalink_url = %v, want %q", data["permalink_url"], "https://app.asana.com/0/12345/67890")
	}
}

func TestCreateTask_OptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var envelope map[string]any
		json.Unmarshal(body, &envelope)
		data := envelope["data"].(map[string]any)

		if data["due_on"] != "2026-03-15" {
			t.Errorf("due_on = %v, want %q", data["due_on"], "2026-03-15")
		}
		if data["notes"] != "Urgent fix needed" {
			t.Errorf("notes = %v, want %q", data["notes"], "Urgent fix needed")
		}
		// Optional fields that were not provided should be absent
		if _, ok := data["due_at"]; ok {
			t.Error("due_at should not be present when not provided")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{"gid": "1", "name": "t", "permalink_url": ""},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.create_task"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.create_task",
		Parameters:  json.RawMessage(`{"project_id":"1","name":"t","notes":"Urgent fix needed","due_on":"2026-03-15"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateTask_MissingProjectID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.create_task"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.create_task",
		Parameters:  json.RawMessage(`{"name":"Test"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateTask_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.create_task"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.create_task",
		Parameters:  json.RawMessage(`{"project_id":"12345"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
