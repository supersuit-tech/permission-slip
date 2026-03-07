package jira

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddComment_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/issue/PROJ-1/comment" {
			t.Errorf("path = %s, want /issue/PROJ-1/comment", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody struct {
			Body map[string]interface{} `json:"body"`
		}
		json.Unmarshal(body, &reqBody)

		if reqBody.Body["type"] != "doc" {
			t.Errorf("body.type = %v, want doc", reqBody.Body["type"])
		}
		if reqBody.Body["version"] != float64(1) {
			t.Errorf("body.version = %v, want 1", reqBody.Body["version"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"id":      "10001",
			"self":    "https://testsite.atlassian.net/rest/api/3/issue/PROJ-1/comment/10001",
			"created": "2024-01-01T00:00:00.000+0000",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["jira.add_comment"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.add_comment",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-1","body":"This is a comment"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]string
	json.Unmarshal(result.Data, &data)
	if data["id"] != "10001" {
		t.Errorf("id = %q, want %q", data["id"], "10001")
	}
}

func TestAddComment_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.add_comment"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing issue_key", `{"body":"test"}`},
		{"missing body", `{"issue_key":"PROJ-1"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "jira.add_comment",
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

func TestPlainTextToADF(t *testing.T) {
	t.Parallel()

	adf := plainTextToADF("Hello world")

	if adf["type"] != "doc" {
		t.Errorf("type = %v, want doc", adf["type"])
	}
	if adf["version"] != 1 {
		t.Errorf("version = %v, want 1", adf["version"])
	}

	content := adf["content"].([]interface{})
	if len(content) != 1 {
		t.Fatalf("content length = %d, want 1", len(content))
	}

	para := content[0].(map[string]interface{})
	if para["type"] != "paragraph" {
		t.Errorf("content[0].type = %v, want paragraph", para["type"])
	}

	textNodes := para["content"].([]interface{})
	textNode := textNodes[0].(map[string]interface{})
	if textNode["text"] != "Hello world" {
		t.Errorf("text = %v, want %q", textNode["text"], "Hello world")
	}
}

func TestPlainTextToADF_Multiline(t *testing.T) {
	t.Parallel()

	adf := plainTextToADF("Line 1\nLine 2\nLine 3")

	content := adf["content"].([]interface{})
	if len(content) != 3 {
		t.Fatalf("content length = %d, want 3 (one paragraph per line)", len(content))
	}

	for i, expected := range []string{"Line 1", "Line 2", "Line 3"} {
		para := content[i].(map[string]interface{})
		textNodes := para["content"].([]interface{})
		textNode := textNodes[0].(map[string]interface{})
		if textNode["text"] != expected {
			t.Errorf("paragraph %d text = %v, want %q", i, textNode["text"], expected)
		}
	}
}

func TestAddComment_WhitespaceOnlyBody(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["jira.add_comment"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "jira.add_comment",
		Parameters:  json.RawMessage(`{"issue_key":"PROJ-1","body":"   "}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error for whitespace-only body, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
