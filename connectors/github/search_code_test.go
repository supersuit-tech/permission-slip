package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSearchCode_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/search/code" {
			t.Errorf("path = %s, want /search/code", r.URL.Path)
		}
		if r.URL.Query().Get("q") != "fmt.Println repo:octocat/hello-world" {
			t.Errorf("q = %q", r.URL.Query().Get("q"))
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"total_count":        1,
			"incomplete_results": false,
			"items": []map[string]any{
				{
					"name":     "main.go",
					"path":     "main.go",
					"html_url": "https://github.com/octocat/hello-world/blob/main/main.go",
					"sha":      "abc123",
					"repository": map[string]any{
						"full_name": "octocat/hello-world",
						"html_url":  "https://github.com/octocat/hello-world",
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.search_code"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.search_code",
		Parameters:  json.RawMessage(`{"q":"fmt.Println repo:octocat/hello-world"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["total_count"] != float64(1) {
		t.Errorf("total_count = %v, want 1", data["total_count"])
	}
}

func TestSearchCode_MissingQuery(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.search_code"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.search_code",
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

func TestSearchCode_InvalidParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.search_code"]

	tests := []struct {
		name   string
		params string
	}{
		{"invalid order", `{"q":"fmt.Println","order":"random"}`},
		{"per_page too large", `{"q":"fmt.Println","per_page":101}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.search_code",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("Execute() expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}
