package confluence

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdatePage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("method = %s, want PUT", r.Method)
		}
		if r.URL.Path != "/pages/98765" {
			t.Errorf("path = %s, want /pages/98765", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		if reqBody["title"] != "Updated Title" {
			t.Errorf("title = %v, want %q", reqBody["title"], "Updated Title")
		}
		version, _ := reqBody["version"].(map[string]interface{})
		if version["number"] != float64(2) {
			t.Errorf("version.number = %v, want 2", version["number"])
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "98765", "title": "Updated Title", "status": "current",
			"version": map[string]interface{}{"number": 2},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["confluence.update_page"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.update_page",
		Parameters:  json.RawMessage(`{"page_id":"98765","title":"Updated Title","version_number":2}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	json.Unmarshal(result.Data, &data)
	if data["title"] != "Updated Title" {
		t.Errorf("title = %v, want %q", data["title"], "Updated Title")
	}
}

func TestUpdatePage_WithBodyAndVersionMessage(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var reqBody map[string]interface{}
		json.Unmarshal(body, &reqBody)

		bodyField, _ := reqBody["body"].(map[string]interface{})
		if bodyField["representation"] != "storage" {
			t.Errorf("body.representation = %v, want storage", bodyField["representation"])
		}
		version, _ := reqBody["version"].(map[string]interface{})
		if version["message"] != "Updated via agent" {
			t.Errorf("version.message = %v, want %q", version["message"], "Updated via agent")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id": "98765", "title": "Page", "status": "current",
			"version": map[string]interface{}{"number": 3, "message": "Updated via agent"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["confluence.update_page"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.update_page",
		Parameters:  json.RawMessage(`{"page_id":"98765","body":"<p>New content</p>","version_number":3,"version_message":"Updated via agent"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestUpdatePage_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["confluence.update_page"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing page_id", `{"title":"Test","version_number":2}`},
		{"missing version_number", `{"page_id":"123","title":"Test"}`},
		{"zero version_number", `{"page_id":"123","title":"Test","version_number":0}`},
		{"no fields to update", `{"page_id":"123","version_number":2}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "confluence.update_page",
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
