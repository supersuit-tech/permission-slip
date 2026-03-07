package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestAddToList_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/marketing/contacts" {
			t.Errorf("got %s %s, want PUT /marketing/contacts", r.Method, r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}

		listIDs, ok := body["list_ids"].([]any)
		if !ok || len(listIDs) != 1 || listIDs[0] != "list_abc" {
			t.Errorf("list_ids = %v, want [list_abc]", body["list_ids"])
		}

		contacts, ok := body["contacts"].([]any)
		if !ok || len(contacts) != 1 {
			t.Errorf("contacts = %v, want 1 contact", body["contacts"])
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{"job_id": "job_789"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.add_to_list"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.add_to_list",
		Parameters:  json.RawMessage(`{"list_id":"list_abc","email":"user@example.com","first_name":"Jane","last_name":"Doe"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["job_id"] != "job_789" {
		t.Errorf("job_id = %v, want job_789", data["job_id"])
	}
	if data["status"] != "accepted" {
		t.Errorf("status = %v, want accepted", data["status"])
	}
}

func TestAddToList_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.add_to_list"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing list_id", params: `{"email":"user@example.com"}`},
		{name: "missing email", params: `{"list_id":"abc"}`},
		{name: "invalid email", params: `{"list_id":"abc","email":"not-an-email"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "sendgrid.add_to_list",
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
