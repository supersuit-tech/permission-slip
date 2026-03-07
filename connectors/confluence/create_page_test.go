package confluence

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreatePage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/pages" {
			t.Errorf("path = %s, want /pages", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		if err := json.Unmarshal(body, &reqBody); err != nil {
			t.Fatalf("unmarshaling request body: %v", err)
		}

		if reqBody["spaceId"] != "123456" {
			t.Errorf("spaceId = %v, want 123456", reqBody["spaceId"])
		}
		if reqBody["title"] != "Test Page" {
			t.Errorf("title = %v, want %q", reqBody["title"], "Test Page")
		}
		if reqBody["status"] != "current" {
			t.Errorf("status = %v, want current", reqBody["status"])
		}
		bodyField, _ := reqBody["body"].(map[string]interface{})
		if bodyField["representation"] != "storage" {
			t.Errorf("body.representation = %v, want storage", bodyField["representation"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "98765",
			"title": "Test Page",
			"status": "current",
			"version": map[string]interface{}{"number": 1},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["confluence.create_page"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.create_page",
		Parameters:  json.RawMessage(`{"space_id":"123456","title":"Test Page","body":"<p>Hello</p>"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["id"] != "98765" {
		t.Errorf("id = %v, want %q", data["id"], "98765")
	}
}

func TestCreatePage_WithOptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		if reqBody["parentId"] != "111" {
			t.Errorf("parentId = %v, want 111", reqBody["parentId"])
		}
		if reqBody["status"] != "draft" {
			t.Errorf("status = %v, want draft", reqBody["status"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "98766", "title": "Draft Page", "status": "draft",
			"version": map[string]interface{}{"number": 1},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["confluence.create_page"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.create_page",
		Parameters:  json.RawMessage(`{"space_id":"123456","title":"Draft Page","body":"<p>Draft</p>","parent_id":"111","status":"draft"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreatePage_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["confluence.create_page"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing space_id", `{"title":"Test","body":"<p>Hi</p>"}`},
		{"missing title", `{"space_id":"123","body":"<p>Hi</p>"}`},
		{"missing body", `{"space_id":"123","title":"Test"}`},
		{"invalid status", `{"space_id":"123","title":"Test","body":"<p>Hi</p>","status":"archived"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "confluence.create_page",
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
