package intercom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateContact_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/contacts" {
			t.Errorf("expected path /contacts, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["email"] != "user@example.com" {
			t.Errorf("expected email user@example.com, got %v", body["email"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(intercomContact{
			Type:  "contact",
			ID:    "abc123",
			Email: "user@example.com",
			Role:  "user",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createContactAction{conn: conn}

	params, _ := json.Marshal(createContactParams{
		Email: "user@example.com",
		Role:  "user",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.create_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data intercomContact
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.ID != "abc123" {
		t.Errorf("expected id abc123, got %q", data.ID)
	}
}

func TestCreateContact_MissingIdentifier(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createContactAction{conn: conn}

	params, _ := json.Marshal(createContactParams{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.create_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing identifier")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateContact_InvalidRole(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createContactAction{conn: conn}

	params, _ := json.Marshal(createContactParams{Email: "x@example.com", Role: "admin"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.create_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid role")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
