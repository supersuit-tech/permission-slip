package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestSearchIssues_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/search/issues" {
			t.Errorf("path = %s, want /search/issues", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "is:open label:bug" {
			t.Errorf("q = %q", r.URL.Query().Get("q"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"total_count":        2,
			"incomplete_results": false,
			"items": []map[string]any{
				{
					"number":   1,
					"title":    "Something broke",
					"state":    "open",
					"html_url": "https://github.com/octocat/hello-world/issues/1",
					"body":     "It broke",
					"user":     map[string]any{"login": "octocat"},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.search_issues"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.search_issues",
		Parameters:  json.RawMessage(`{"q":"is:open label:bug"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["total_count"] != float64(2) {
		t.Errorf("total_count = %v, want 2", data["total_count"])
	}
}

func TestSearchIssues_WithSortOrder(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sort") != "created" {
			t.Errorf("sort = %q, want created", r.URL.Query().Get("sort"))
		}
		if r.URL.Query().Get("order") != "desc" {
			t.Errorf("order = %q, want desc", r.URL.Query().Get("order"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"total_count":        0,
			"incomplete_results": false,
			"items":              []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.search_issues"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.search_issues",
		Parameters:  json.RawMessage(`{"q":"is:open","sort":"created","order":"desc"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSearchIssues_MissingQuery(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.search_issues"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.search_issues",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
