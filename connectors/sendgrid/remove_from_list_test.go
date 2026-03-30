package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestRemoveFromList_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("method = %s, want DELETE", r.Method)
		}
		wantPath := "/marketing/lists/list_abc/contacts"
		if r.URL.Path != wantPath {
			t.Errorf("path = %s, want %s", r.URL.Path, wantPath)
		}
		if got := r.URL.Query().Get("contact_ids"); got != "contact_xyz" {
			t.Errorf("contact_ids = %q, want contact_xyz", got)
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{"job_id": "job_del_123"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.remove_from_list"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.remove_from_list",
		Parameters:  json.RawMessage(`{"list_id":"list_abc","contact_id":"contact_xyz"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["job_id"] != "job_del_123" {
		t.Errorf("job_id = %v, want job_del_123", data["job_id"])
	}
	if data["status"] != "accepted" {
		t.Errorf("status = %v, want accepted", data["status"])
	}
}

func TestRemoveFromList_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.remove_from_list"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing list_id", params: `{"contact_id":"abc"}`},
		{name: "missing contact_id", params: `{"list_id":"abc"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "sendgrid.remove_from_list",
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
