package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateCompany_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/companies" {
			t.Errorf("expected path /crm/v3/objects/companies, got %s", r.URL.Path)
		}

		var body hubspotObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body.Properties["name"] != "Acme Corp" {
			t.Errorf("expected name Acme Corp, got %q", body.Properties["name"])
		}
		if body.Properties["domain"] != "acme.com" {
			t.Errorf("expected domain acme.com, got %q", body.Properties["domain"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(hubspotObjectResponse{
			ID:         "500",
			Properties: body.Properties,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createCompanyAction{conn: conn}

	params, _ := json.Marshal(createCompanyParams{
		Name:   "Acme Corp",
		Domain: "acme.com",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_company",
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
	if data.ID != "500" {
		t.Errorf("expected id 500, got %q", data.ID)
	}
}

func TestCreateCompany_MissingName(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCompanyAction{conn: conn}

	params, _ := json.Marshal(createCompanyParams{Domain: "acme.com"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.create_company",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
