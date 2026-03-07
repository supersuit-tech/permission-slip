package twilio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetCall_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		wantPath := "/Accounts/" + testAccountSID + "/Calls/CA1234567890abcdef1234567890abcdef.json"
		if got := r.URL.Path; got != wantPath {
			t.Errorf("path = %s, want %s", got, wantPath)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"sid":        "CA1234567890abcdef1234567890abcdef",
			"status":     "completed",
			"to":         "+15551234567",
			"from":       "+15559876543",
			"duration":   "42",
			"direction":  "outbound-api",
			"start_time": "Mon, 01 Jan 2024 12:00:00 +0000",
			"end_time":   "Mon, 01 Jan 2024 12:00:42 +0000",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.get_call"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.get_call",
		Parameters:  json.RawMessage(`{"call_sid":"CA1234567890abcdef1234567890abcdef"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["status"] != "completed" {
		t.Errorf("status = %v, want completed", data["status"])
	}
	if data["duration"] != "42" {
		t.Errorf("duration = %v, want 42", data["duration"])
	}
}

func TestGetCall_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["twilio.get_call"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing call_sid", params: `{}`},
		{name: "empty call_sid", params: `{"call_sid":""}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "twilio.get_call",
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
