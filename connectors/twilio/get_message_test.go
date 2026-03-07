package twilio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetMessage_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		wantPath := "/Accounts/" + testAccountSID + "/Messages/SM1234567890abcdef1234567890abcdef.json"
		if got := r.URL.Path; got != wantPath {
			t.Errorf("path = %s, want %s", got, wantPath)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != testAccountSID || pass != testAuthToken {
			t.Errorf("BasicAuth = (%q, %q, %v), want (%q, %q, true)", user, pass, ok, testAccountSID, testAuthToken)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"sid":       "SM1234567890abcdef1234567890abcdef",
			"status":    "delivered",
			"to":        "+15551234567",
			"from":      "+15559876543",
			"body":      "Hello",
			"date_sent": "Mon, 01 Jan 2024 12:00:00 +0000",
			"direction": "outbound-api",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.get_message"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.get_message",
		Parameters:  json.RawMessage(`{"message_sid":"SM1234567890abcdef1234567890abcdef"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "delivered" {
		t.Errorf("status = %v, want delivered", data["status"])
	}
	if data["direction"] != "outbound-api" {
		t.Errorf("direction = %v, want outbound-api", data["direction"])
	}
}

func TestGetMessage_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"code":    20404,
			"message": "The requested resource was not found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.get_message"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.get_message",
		Parameters:  json.RawMessage(`{"message_sid":"SM0000000000000000000000000000000000"}`),
		Credentials: validCreds(),
	})

	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestGetMessage_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["twilio.get_message"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing message_sid", params: `{}`},
		{name: "empty message_sid", params: `{"message_sid":""}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "twilio.get_message",
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
