package twilio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSendWhatsApp_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}

		if err := r.ParseForm(); err != nil {
			t.Fatalf("parsing form: %v", err)
		}
		if got := r.PostFormValue("To"); got != "whatsapp:+15551234567" {
			t.Errorf("To = %q, want %q", got, "whatsapp:+15551234567")
		}
		if got := r.PostFormValue("From"); got != "whatsapp:+15559876543" {
			t.Errorf("From = %q, want %q", got, "whatsapp:+15559876543")
		}
		if got := r.PostFormValue("Body"); got != "Hello via WhatsApp" {
			t.Errorf("Body = %q, want %q", got, "Hello via WhatsApp")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"sid":    "SM1234567890abcdef1234567890abcdef",
			"status": "queued",
			"to":     "whatsapp:+15551234567",
			"from":   "whatsapp:+15559876543",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.send_whatsapp"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.send_whatsapp",
		Parameters:  json.RawMessage(`{"to":"+15551234567","from":"+15559876543","body":"Hello via WhatsApp"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["sid"] != "SM1234567890abcdef1234567890abcdef" {
		t.Errorf("sid = %v, want SM1234567890abcdef1234567890abcdef", data["sid"])
	}
}

func TestSendWhatsApp_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["twilio.send_whatsapp"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing to", params: `{"from":"+15559876543","body":"Hello"}`},
		{name: "missing from", params: `{"to":"+15551234567","body":"Hello"}`},
		{name: "missing body", params: `{"to":"+15551234567","from":"+15559876543"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "twilio.send_whatsapp",
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
