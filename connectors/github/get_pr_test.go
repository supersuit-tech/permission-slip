package github

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetPR_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/repos/o/r/pulls/12" {
			t.Errorf("path = %s, want /repos/o/r/pulls/12", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"number": 12, "title": "feat", "state": "open",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.get_pr"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.get_pr",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","pull_number":12}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["title"] != "feat" {
		t.Errorf("title = %v, want feat", data["title"])
	}
}

func TestGetPR_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.get_pr"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing owner", `{"repo":"r","pull_number":1}`},
		{"missing pull_number", `{"owner":"o","repo":"r"}`},
		{"zero pull_number", `{"owner":"o","repo":"r","pull_number":0}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.get_pr",
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
