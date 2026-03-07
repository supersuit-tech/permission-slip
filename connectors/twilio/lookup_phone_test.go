package twilio

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestLookupPhone_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("method = %s, want GET", r.Method)
		}
		wantPath := "/PhoneNumbers/+15551234567"
		if got := r.URL.Path; got != wantPath {
			t.Errorf("path = %s, want %s", got, wantPath)
		}

		user, pass, ok := r.BasicAuth()
		if !ok || user != testAccountSID || pass != testAuthToken {
			t.Errorf("BasicAuth = (%q, %q, %v), want (%q, %q, true)", user, pass, ok, testAccountSID, testAuthToken)
		}

		json.NewEncoder(w).Encode(map[string]any{
			"phone_number":         "+15551234567",
			"country_code":         "US",
			"national_format":      "(555) 123-4567",
			"valid":                true,
			"calling_country_code": "1",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["twilio.lookup_phone"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "twilio.lookup_phone",
		Parameters:  json.RawMessage(`{"phone_number":"+15551234567"}`),
		Credentials: validCreds(),
	})

	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["country_code"] != "US" {
		t.Errorf("country_code = %v, want US", data["country_code"])
	}
	if data["valid"] != true {
		t.Errorf("valid = %v, want true", data["valid"])
	}
}

func TestLookupPhone_MissingParams(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["twilio.lookup_phone"]

	tests := []struct {
		name   string
		params string
	}{
		{name: "missing phone_number", params: `{}`},
		{name: "empty phone_number", params: `{"phone_number":""}`},
		{name: "invalid format", params: `{"phone_number":"5551234567"}`},
		{name: "invalid JSON", params: `{invalid}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "twilio.lookup_phone",
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
