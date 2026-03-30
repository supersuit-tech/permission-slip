package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateContact_Success(t *testing.T) {
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

		// No list_ids — this is the key difference from add_to_list.
		if _, hasListIDs := body["list_ids"]; hasListIDs {
			t.Error("create_contact should not send list_ids")
		}

		contacts, ok := body["contacts"].([]any)
		if !ok || len(contacts) != 1 {
			t.Errorf("contacts = %v, want 1 contact", body["contacts"])
		}

		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]any{"job_id": "job_contact_123"})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.create_contact"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.create_contact",
		Parameters:  json.RawMessage(`{"email":"contact@example.com","first_name":"Jane","last_name":"Doe","city":"Portland","country":"US"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["job_id"] != "job_contact_123" {
		t.Errorf("job_id = %v, want job_contact_123", data["job_id"])
	}
	if data["status"] != "accepted" {
		t.Errorf("status = %v, want accepted", data["status"])
	}
}

func TestCreateContact_ValidationErrors(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.create_contact"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing email", params: `{"first_name":"Jane"}`},
		{name: "invalid email", params: `{"email":"not-an-email"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "sendgrid.create_contact",
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
