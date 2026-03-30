package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGetContact_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/crm/v3/objects/contacts/123") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(hubspotObjectResponse{
			ID: "123",
			Properties: map[string]string{
				"email":     "alice@example.com",
				"firstname": "Alice",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getContactAction{conn: conn}

	params, _ := json.Marshal(getContactParams{ContactID: "123"})
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.get_contact",
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
	if data.ID != "123" {
		t.Errorf("expected id 123, got %q", data.ID)
	}
	if data.Properties["email"] != "alice@example.com" {
		t.Errorf("expected email alice@example.com, got %q", data.Properties["email"])
	}
}

func TestGetContact_MissingID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getContactAction{conn: conn}

	params, _ := json.Marshal(getContactParams{})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.get_contact",
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

func TestGetContact_NonNumericID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getContactAction{conn: conn}

	params, _ := json.Marshal(getContactParams{ContactID: "abc"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.get_contact",
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
