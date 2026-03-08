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

		// X-Message-Id is returned by SendGrid for delivery tracking.
		w.Header().Set("X-Message-Id", "abc123xyz")
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
	if data["status"] != "accepted" {
		t.Errorf("status = %v, want accepted", data["status"])
	}
	if data["to"] != "user@example.com" {
		t.Errorf("to = %v, want user@example.com", data["to"])
	}
	if data["message_id"] != "abc123xyz" {
		t.Errorf("message_id = %v, want abc123xyz", data["message_id"])
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

func TestSendTransactionalEmail_WithCCBCCAndCategories(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decoding body: %v", err)
			return
		}

		perz := body["personalizations"].([]any)
		first := perz[0].(map[string]any)

		// Verify CC is present in personalization.
		cc, ok := first["cc"].([]any)
		if !ok || len(cc) != 1 {
			t.Errorf("cc = %v, want 1 address", first["cc"])
		} else if cc[0].(map[string]any)["email"] != "cc@example.com" {
			t.Errorf("cc[0].email = %v, want cc@example.com", cc[0])
		}

		// Verify BCC is present in personalization.
		bcc, ok := first["bcc"].([]any)
		if !ok || len(bcc) != 1 {
			t.Errorf("bcc = %v, want 1 address", first["bcc"])
		} else if bcc[0].(map[string]any)["email"] != "bcc@example.com" {
			t.Errorf("bcc[0].email = %v, want bcc@example.com", bcc[0])
		}

		// Verify categories are present at the top level.
		cats, ok := body["categories"].([]any)
		if !ok || len(cats) != 2 {
			t.Errorf("categories = %v, want 2 items", body["categories"])
		}

		w.WriteHeader(http.StatusAccepted)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["sendgrid.send_transactional_email"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "sendgrid.send_transactional_email",
		Parameters: json.RawMessage(`{
			"to":"user@example.com",
			"from":"sender@example.com",
			"subject":"Hi",
			"html_content":"<p>Hi</p>",
			"cc":["cc@example.com"],
			"bcc":["bcc@example.com"],
			"categories":["welcome","onboarding"]
		}`),
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
		{name: "invalid cc email", params: `{"to":"user@example.com","from":"sender@example.com","subject":"Hi","html_content":"<p>Hi</p>","cc":["not-an-email"]}`},
		{name: "invalid bcc email", params: `{"to":"user@example.com","from":"sender@example.com","subject":"Hi","html_content":"<p>Hi</p>","bcc":["bad"]}`},
		{name: "too many categories", params: `{"to":"user@example.com","from":"sender@example.com","subject":"Hi","html_content":"<p>Hi</p>","categories":["a","b","c","d","e","f","g","h","i","j","k"]}`},
		{name: "dynamic_template_data without template_id", params: `{"to":"user@example.com","from":"sender@example.com","subject":"Hi","html_content":"<p>Hi</p>","dynamic_template_data":{"key":"value"}}`},
		{name: "subject too long", params: `{"to":"user@example.com","from":"sender@example.com","subject":"` + string(make([]byte, 999)) + `","html_content":"<p>Hi</p>"}`},
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
