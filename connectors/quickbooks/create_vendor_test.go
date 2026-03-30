package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateVendor_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/company/1234567890/vendor" {
			t.Errorf("path = %s, want /v3/company/1234567890/vendor", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["DisplayName"] != "Acme Supplies" {
			t.Errorf("DisplayName = %v, want Acme Supplies", body["DisplayName"])
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Vendor": map[string]any{
				"Id":          "100",
				"DisplayName": "Acme Supplies",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.create_vendor"]
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_vendor",
		Parameters:  json.RawMessage(`{"display_name": "Acme Supplies", "email": "billing@acme.com"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshal result: %v", err)
	}
	if data["Id"] != "100" {
		t.Errorf("Id = %v, want 100", data["Id"])
	}
}

func TestCreateVendor_MissingDisplayName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_vendor"]

	tests := []struct {
		name   string
		params string
	}{
		{"missing display_name", `{"email": "vendor@example.com"}`},
		{"empty display_name", `{"display_name": ""}`},
		{"invalid JSON", `{bad}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := action.Execute(t.Context(), connectors.ActionRequest{
				ActionType:  "quickbooks.create_vendor",
				Parameters:  json.RawMessage(tt.params),
				Credentials: validCreds(),
			})
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestCreateVendor_WithOptionalFields(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)

		if body["GivenName"] != "John" {
			t.Errorf("GivenName = %v, want John", body["GivenName"])
		}
		if body["CompanyName"] != "Acme Corp" {
			t.Errorf("CompanyName = %v, want Acme Corp", body["CompanyName"])
		}

		email, ok := body["PrimaryEmailAddr"].(map[string]any)
		if !ok || email["Address"] != "john@acme.com" {
			t.Errorf("PrimaryEmailAddr = %v", body["PrimaryEmailAddr"])
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Vendor": map[string]any{"Id": "101"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.create_vendor"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "quickbooks.create_vendor",
		Parameters: json.RawMessage(`{
			"display_name": "John Acme",
			"given_name": "John",
			"company_name": "Acme Corp",
			"email": "john@acme.com"
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateVendor_InvalidEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_vendor"]
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_vendor",
		Parameters:  json.RawMessage(`{"display_name": "Acme", "email": "not-an-email"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid email, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
