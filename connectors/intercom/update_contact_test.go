package intercom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateContact_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if r.URL.Path != "/contacts/abc123" {
			t.Errorf("expected path /contacts/abc123, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(intercomContact{
			Type:  "contact",
			ID:    "abc123",
			Email: "updated@example.com",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateContactAction{conn: conn}

	params, _ := json.Marshal(updateContactParams{
		ContactID: "abc123",
		Email:     "updated@example.com",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.update_contact",
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

func TestUpdateContact_MissingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateContactAction{conn: conn}

	params, _ := json.Marshal(updateContactParams{Email: "x@example.com"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.update_contact",
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

func TestUpdateContact_NoFieldsToUpdate(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateContactAction{conn: conn}

	params, _ := json.Marshal(updateContactParams{ContactID: "abc123"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.update_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for no fields to update")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
