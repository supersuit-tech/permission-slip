package jira

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListSprints_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/board/42/sprint" {
			t.Errorf("path = %s, want /board/42/sprint", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"values": []map[string]interface{}{
				{"id": 1, "name": "Sprint 1", "state": "active"},
				{"id": 2, "name": "Sprint 2", "state": "future"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.list_sprints"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.list_sprints",
		Parameters:  json.RawMessage(`{"board_id":42}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	json.Unmarshal(result.Data, &data)
	if data["total_count"] != float64(2) {
		t.Errorf("total_count = %v, want 2", data["total_count"])
	}
	sprints, ok := data["sprints"].([]interface{})
	if !ok {
		t.Fatal("expected sprints array")
	}
	if len(sprints) != 2 {
		t.Errorf("got %d sprints, want 2", len(sprints))
	}
}

func TestListSprints_WithState(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "active" {
			t.Errorf("state = %q, want active", r.URL.Query().Get("state"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"values": []interface{}{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.list_sprints"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.list_sprints",
		Parameters:  json.RawMessage(`{"board_id":42,"state":"active"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestListSprints_InvalidState(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.list_sprints"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.list_sprints",
		Parameters:  json.RawMessage(`{"board_id":42,"state":"invalid"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestListSprints_MissingBoardID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.list_sprints"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.list_sprints",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing board_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
