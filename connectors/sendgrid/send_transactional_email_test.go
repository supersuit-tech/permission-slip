package sendgrid

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSendTransactionalEmail_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/mail/send" {
			t.Errorf("got %s %s, want POST /mail/send", r.Method, r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}

		from, ok := body["from"].(map[string]any)
		if !ok || from["email"] != "sender@example.com" {
			t.Errorf("from.email = %v, want sender@example.com", body["from"])
		}

		perz, ok := body["personalizations"].([]any)
		if !ok || len(perz) == 0 {
			t.Errorf("personalizations missing or empty")
		}

		if body["subject"] != "Hello" {
			t.Errorf("subject = %v, want Hello", body["subject"])
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.send_transactional_email"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.send_transactional_email",
		Parameters:  json.RawMessage(`{"to":"user@example.com","from":"sender@example.com","subject":"Hello","html_content":"<p>Hi</p>"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "sent" {
		t.Errorf("status = %v, want sent", data["status"])
	}
	if data["to"] != "user@example.com" {
		t.Errorf("to = %v, want user@example.com", data["to"])
	}
}

func TestSendTransactionalEmail_WithTemplate(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}

		if body["template_id"] != "d-abc123" {
			t.Errorf("template_id = %v, want d-abc123", body["template_id"])
		}

		perz := body["personalizations"].([]any)
		first := perz[0].(map[string]any)
		dynData, ok := first["dynamic_template_data"].(map[string]any)
		if !ok {
			t.Errorf("dynamic_template_data missing from personalization")
		} else if dynData["first_name"] != "Jane" {
			t.Errorf("dynamic_template_data.first_name = %v, want Jane", dynData["first_name"])
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.send_transactional_email"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "sendgrid.send_transactional_email",
		Parameters:  json.RawMessage(`{"to":"user@example.com","from":"sender@example.com","template_id":"d-abc123","dynamic_template_data":{"first_name":"Jane"}}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestSendTransactionalEmail_ValidationErrors(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["sendgrid.send_transactional_email"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing to", params: `{"from":"sender@example.com","subject":"Hi","html_content":"<p>Hi</p>"}`},
		{name: "missing from", params: `{"to":"user@example.com","subject":"Hi","html_content":"<p>Hi</p>"}`},
		{name: "invalid to email", params: `{"to":"not-an-email","from":"sender@example.com","subject":"Hi","html_content":"<p>Hi</p>"}`},
		{name: "invalid from email", params: `{"to":"user@example.com","from":"not-an-email","subject":"Hi","html_content":"<p>Hi</p>"}`},
		{name: "missing subject without template", params: `{"to":"user@example.com","from":"sender@example.com","html_content":"<p>Hi</p>"}`},
		{name: "missing content without template", params: `{"to":"user@example.com","from":"sender@example.com","subject":"Hi"}`},
		{name: "invalid reply_to", params: `{"to":"user@example.com","from":"sender@example.com","subject":"Hi","html_content":"<p>Hi</p>","reply_to":"bad"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "sendgrid.send_transactional_email",
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
