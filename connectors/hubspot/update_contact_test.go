package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateContact_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/contacts/501" {
			t.Errorf("expected path /crm/v3/objects/contacts/501, got %s", r.URL.Path)
		}

		var body hubspotObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Properties["phone"] != "555-1234" {
			t.Errorf("expected phone 555-1234, got %q", body.Properties["phone"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id": "501",
			"properties": map[string]string{
				"email": "jane@example.com",
				"phone": "555-1234",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateContactAction{conn: conn}

	params, _ := json.Marshal(updateContactParams{
		ContactID:  "501",
		Properties: map[string]string{"phone": "555-1234"},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_contact",
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

func TestUpdateContact_MissingContactID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateContactAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"properties": map[string]string{"phone": "555-1234"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing contact_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateContact_EmptyProperties(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateContactAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"contact_id": "501",
		"properties": map[string]string{},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for empty properties")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateContact_PathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateContactAction{conn: conn}

	params, _ := json.Marshal(updateContactParams{
		ContactID:  "../../admin",
		Properties: map[string]string{"phone": "555-1234"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-numeric contact_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateContact_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{
			"status":   "error",
			"category": "OBJECT_NOT_FOUND",
			"message":  "Contact not found",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateContactAction{conn: conn}

	params, _ := json.Marshal(updateContactParams{
		ContactID:  "999",
		Properties: map[string]string{"phone": "555-1234"},
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for not found contact")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for OBJECT_NOT_FOUND, got: %T", err)
	}
}
