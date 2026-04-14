package github

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestRemoveReviewer_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		if r.URL.Path != "/repos/o/r/pulls/8/requested_reviewers" {
			t.Errorf("path = %s, want /repos/o/r/pulls/8/requested_reviewers", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			Reviewers []string `json:"reviewers"`
		}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling: %v", err)
		}
		want := []string{"alice", "bob"}
		if len(reqBody.Reviewers) != len(want) || reqBody.Reviewers[0] != want[0] || reqBody.Reviewers[1] != want[1] {
			t.Errorf("reviewers = %v, want %v", reqBody.Reviewers, want)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{"number": 8})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["github.remove_reviewer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "github.remove_reviewer",
		Parameters:  json.RawMessage(`{"owner":"o","repo":"r","pull_number":8,"reviewers":["alice","bob"]}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestRemoveReviewer_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["github.remove_reviewer"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing reviewers", `{"owner":"o","repo":"r","pull_number":1}`},
		{"empty reviewers", `{"owner":"o","repo":"r","pull_number":1,"reviewers":[]}`},
		{"empty string in reviewers", `{"owner":"o","repo":"r","pull_number":1,"reviewers":["alice",""]}`},
		{"zero pull_number", `{"owner":"o","repo":"r","pull_number":0,"reviewers":["alice"]}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "github.remove_reviewer",
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
