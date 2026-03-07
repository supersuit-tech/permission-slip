package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListLists_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/marketing/lists" {
			t.Errorf("path = %s, want /marketing/lists", r.URL.Path)
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"result": []map[string]any{
				{
					"id":            "list_abc",
					"name":          "Newsletter Subscribers",
					"contact_count": 5000,
				},
				{
					"id":            "list_xyz",
					"name":          "Beta Users",
					"contact_count": 250,
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.list_lists"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.list_lists",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["count"] != float64(2) {
		t.Errorf("count = %v, want 2", data["count"])
	}
	lists, ok := data["lists"].([]any)
	if !ok {
		t.Fatal("lists not present in result")
	}
	if len(lists) != 2 {
		t.Errorf("lists length = %d, want 2", len(lists))
	}
	first := lists[0].(map[string]any)
	if first["name"] != "Newsletter Subscribers" {
		t.Errorf("first list name = %v, want Newsletter Subscribers", first["name"])
	}
	if first["contact_count"] != float64(5000) {
		t.Errorf("first list contact_count = %v, want 5000", first["contact_count"])
	}
}

func TestListLists_Empty(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"result": []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.list_lists"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.list_lists",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["count"] != float64(0) {
		t.Errorf("count = %v, want 0", data["count"])
	}
}
