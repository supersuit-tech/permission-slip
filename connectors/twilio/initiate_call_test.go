package twilio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestInitiateCall_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if got := r.URL.Path; got != "/Accounts/"+testAccountSID+"/Calls.json" {
			t.Errorf("path = %s, want /Accounts/%s/Calls.json", got, testAccountSID)
		}

		if err := r.ParseForm(); err != nil {
			t.Errorf("parsing form: %v", err)
			return
		}
		if got := r.PostFormValue("To"); got != "+15551234567" {
			t.Errorf("To = %q, want %q", got, "+15551234567")
		}
		if got := r.PostFormValue("From"); got != "+15559876543" {
			t.Errorf("From = %q, want %q", got, "+15559876543")
		}
		if got := r.PostFormValue("Twiml"); got != "<Response><Say>Hello</Say></Response>" {
			t.Errorf("Twiml = %q, want %q", got, "<Response><Say>Hello</Say></Response>")
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"sid":    "CA1234567890abcdef1234567890abcdef",
			"status": "queued",
			"to":     "+15551234567",
			"from":   "+15559876543",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.initiate_call"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.initiate_call",
		Parameters:  json.RawMessage(`{"to":"+15551234567","from":"+15559876543","twiml":"<Response><Say>Hello</Say></Response>"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["sid"] != "CA1234567890abcdef1234567890abcdef" {
		t.Errorf("sid = %v, want CA1234567890abcdef1234567890abcdef", data["sid"])
	}
	if data["status"] != "queued" {
		t.Errorf("status = %v, want queued", data["status"])
	}
}

func TestInitiateCall_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["twilio.initiate_call"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing to", params: `{"from":"+15559876543","twiml":"<Response><Say>Hi</Say></Response>"}`},
		{name: "missing from", params: `{"to":"+15551234567","twiml":"<Response><Say>Hi</Say></Response>"}`},
		{name: "missing twiml", params: `{"to":"+15551234567","from":"+15559876543"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "twilio.initiate_call",
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
