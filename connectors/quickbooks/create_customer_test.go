package quickbooks

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateCustomer_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/v3/company/1234567890/customer" {
			t.Errorf("path = %s, want /v3/company/1234567890/customer", r.URL.Path)
		}

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if body["DisplayName"] != "Acme Corp" {
			t.Errorf("DisplayName = %v, want Acme Corp", body["DisplayName"])
		}
		if body["GivenName"] != "Jane" {
			t.Errorf("GivenName = %v, want Jane", body["GivenName"])
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Customer": map[string]any{
				"Id":          "42",
				"DisplayName": "Acme Corp",
				"GivenName":   "Jane",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.create_customer"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType: "quickbooks.create_customer",
		Parameters: json.RawMessage(`{
			"display_name": "Acme Corp",
			"given_name": "Jane",
			"family_name": "Doe",
			"email": "jane@acme.com",
			"phone": "555-1234"
		}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["Id"] != "42" {
		t.Errorf("Id = %v, want 42", data["Id"])
	}
	if data["DisplayName"] != "Acme Corp" {
		t.Errorf("DisplayName = %v, want Acme Corp", data["DisplayName"])
	}
}

func TestCreateCustomer_DisplayNameOnly(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		if _, ok := body["GivenName"]; ok {
			t.Error("GivenName should not be present when not provided")
		}

		json.NewEncoder(w).Encode(map[string]any{
			"Customer": map[string]any{"Id": "43", "DisplayName": "Bob"},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["quickbooks.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_customer",
		Parameters:  json.RawMessage(`{"display_name":"Bob"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
}

func TestCreateCustomer_MissingDisplayName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_customer",
		Parameters:  json.RawMessage(`{"email":"test@example.com"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCreateCustomer_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["quickbooks.create_customer"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "quickbooks.create_customer",
		Parameters:  json.RawMessage(`{invalid}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}
