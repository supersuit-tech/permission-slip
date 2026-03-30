package confluence

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestAddComment_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/pages/98765/footer-comments" {
			t.Errorf("path = %s, want /pages/98765/footer-comments", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		bodyField, _ := reqBody["body"].(map[string]interface{})
		if bodyField["representation"] != "storage" {
			t.Errorf("body.representation = %v, want storage", bodyField["representation"])
		}
		if bodyField["value"] != "<p>Great page!</p>" {
			t.Errorf("body.value = %v, want %q", bodyField["value"], "<p>Great page!</p>")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"id": "comment-1"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["confluence.add_comment"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.add_comment",
		Parameters:  json.RawMessage(`{"page_id":"98765","body":"<p>Great page!</p>"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]string
	json.Unmarshal(result.Data, &data)
	if data["id"] != "comment-1" {
		t.Errorf("id = %q, want %q", data["id"], "comment-1")
	}
}

func TestAddComment_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["confluence.add_comment"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing page_id", `{"body":"<p>Hi</p>"}`},
		{"missing body", `{"page_id":"123"}`},
		{"empty body", `{"page_id":"123","body":"   "}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "confluence.add_comment",
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
