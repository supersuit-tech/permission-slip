package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDeleteContact_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/contacts/123" {
			t.Errorf("expected path /crm/v3/objects/contacts/123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &deleteContactAction{conn: conn}

	params, _ := json.Marshal(deleteContactParams{ContactID: "123"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.delete_contact",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["contact_id"] != "123" {
		t.Errorf("expected contact_id 123, got %v", data["contact_id"])
	}
	if data["archived"] != true {
		t.Errorf("expected archived=true, got %v", data["archived"])
	}
}

func TestDeleteContact_MissingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteContactAction{conn: conn}

	params, _ := json.Marshal(deleteContactParams{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.delete_contact",
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

func TestDeleteContact_NonNumericID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &deleteContactAction{conn: conn}

	params, _ := json.Marshal(deleteContactParams{ContactID: "not-a-number"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.delete_contact",
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
