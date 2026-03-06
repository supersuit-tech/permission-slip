package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateContact_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/contacts" {
			t.Errorf("expected path /crm/v3/objects/contacts, got %s", r.URL.Path)
		}

		var body hubspotObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Properties["email"] != "jane@example.com" {
			t.Errorf("expected email jane@example.com, got %q", body.Properties["email"])
		}
		if body.Properties["firstname"] != "Jane" {
			t.Errorf("expected firstname Jane, got %q", body.Properties["firstname"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{
			"id": "501",
			"properties": map[string]string{
				"email":     "jane@example.com",
				"firstname": "Jane",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createContactAction{conn: conn}

	params, _ := json.Marshal(createContactParams{
		Email:     "jane@example.com",
		FirstName: "Jane",
		LastName:  "Doe",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data hubspotObjectResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.ID != "501" {
		t.Errorf("expected id 501, got %q", data.ID)
	}
}

func TestCreateContact_WithAdditionalProperties(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body hubspotObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		// Explicit field should override additional properties.
		if body.Properties["email"] != "jane@example.com" {
			t.Errorf("expected email jane@example.com, got %q", body.Properties["email"])
		}
		if body.Properties["custom_field"] != "custom_value" {
			t.Errorf("expected custom_field custom_value, got %q", body.Properties["custom_field"])
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]any{"id": "502", "properties": body.Properties})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createContactAction{conn: conn}

	params, _ := json.Marshal(createContactParams{
		Email:      "jane@example.com",
		Properties: map[string]string{"custom_field": "custom_value"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateContact_MissingEmail(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createContactAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"firstname": "Jane"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing email")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateContact_APIError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusConflict)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "error",
			"category": "CONTACT_EXISTS",
			"message":  "Contact already exists",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createContactAction{conn: conn}

	params, _ := json.Marshal(createContactParams{Email: "existing@example.com"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for duplicate contact")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for CONTACT_EXISTS, got: %T", err)
	}
}

func TestCreateContact_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createContactAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_contact",
		Parameters:  json.RawMessage(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
