package confluence

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetPage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/pages/98765" {
			t.Errorf("path = %s, want /pages/98765", r.URL.Path)
		}
		if got := r.URL.Query().Get("body-format"); got != "storage" {
			t.Errorf("body-format = %q, want storage", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"id":    "98765",
			"title": "My Page",
			"body": map[string]interface{}{
				"storage": map[string]interface{}{
					"representation": "storage",
					"value":          "<p>Hello world</p>",
				},
			},
			"version": map[string]interface{}{"number": 3},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["confluence.get_page"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.get_page",
		Parameters:  json.RawMessage(`{"page_id":"98765"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]interface{}
	json.Unmarshal(result.Data, &data)
	if data["id"] != "98765" {
		t.Errorf("id = %v, want %q", data["id"], "98765")
	}
	if data["title"] != "My Page" {
		t.Errorf("title = %v, want %q", data["title"], "My Page")
	}
}

func TestGetPage_CustomBodyFormat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("body-format"); got != "atlas_doc_format" {
			t.Errorf("body-format = %q, want atlas_doc_format", got)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "98765", "title": "Page"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["confluence.get_page"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.get_page",
		Parameters:  json.RawMessage(`{"page_id":"98765","body_format":"atlas_doc_format"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetPage_WhitespacePaddedBodyFormat(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Query().Get("body-format"); got != "view" {
			t.Errorf("body-format = %q, want view", got)
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"id": "98765", "title": "Page"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["confluence.get_page"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.get_page",
		Parameters:  json.RawMessage(`{"page_id":"98765","body_format":" view "}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestGetPage_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["confluence.get_page"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing page_id", `{}`},
		{"empty page_id", `{"page_id":""}`},
		{"invalid body_format", `{"page_id":"123","body_format":"invalid"}`},
		{"invalid JSON", `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "confluence.get_page",
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
