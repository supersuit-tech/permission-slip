package asana

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateSection_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/projects/proj1/sections" {
			t.Errorf("path = %s, want /projects/proj1/sections", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		var envelope map[string]any
		json.Unmarshal(body, &envelope)
		data := envelope["data"].(map[string]any)
		if data["name"] != "Review" {
			t.Errorf("name = %v, want Review", data["name"])
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"data": map[string]any{
				"gid":  "sec1",
				"name": "Review",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["asana.create_section"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.create_section",
		Parameters:  json.RawMessage(`{"project_id":"proj1","name":"Review"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	json.Unmarshal(result.Data, &data)
	if data["name"] != "Review" {
		t.Errorf("name = %v, want Review", data["name"])
	}
}

func TestCreateSection_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["asana.create_section"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "asana.create_section",
		Parameters:  json.RawMessage(`{"project_id":"proj1"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
