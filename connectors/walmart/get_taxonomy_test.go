package walmart

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetTaxonomy_Success(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		if r.URL.Path != "/taxonomy" {
			t.Errorf("path = %s, want /taxonomy", r.URL.Path)
		}
		if got := r.URL.Query().Get("format"); got != "json" {
			t.Errorf("format = %q, want %q", got, "json")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"categories": []map[string]any{
				{
					"id":   "0",
					"name": "All Departments",
					"children": []map[string]any{
						{"id": "5438", "name": "Electronics"},
						{"id": "976759", "name": "Grocery"},
					},
				},
			},
		})
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["walmart.get_taxonomy"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "walmart.get_taxonomy",
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
	if _, ok := data["categories"]; !ok {
		t.Error("expected categories in response")
	}
}

func TestGetTaxonomy_APIError(t *testing.T) {
	t.Parallel()

	srv := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"message":"Internal Server Error"}`))
	})
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["walmart.get_taxonomy"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "walmart.get_taxonomy",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
