package hubspot

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateCompany_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		if r.URL.Path != "/crm/v3/objects/companies/500" {
			t.Errorf("expected path /crm/v3/objects/companies/500, got %s", r.URL.Path)
		}

		var body hubspotObjectRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(hubspotObjectResponse{
			ID:         "500",
			Properties: body.Properties,
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateCompanyAction{conn: conn}

	params, _ := json.Marshal(updateCompanyParams{
		CompanyID:  "500",
		Properties: map[string]string{"phone": "+1-555-0100"},
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_company",
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

func TestUpdateCompany_MissingProperties(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateCompanyAction{conn: conn}

	params, _ := json.Marshal(updateCompanyParams{CompanyID: "500"})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "hubspot.update_company",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing properties")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
