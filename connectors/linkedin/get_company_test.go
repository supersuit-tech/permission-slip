package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetCompany_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/organizations/") {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(organizationResponse{
			ID: "12345",
			Name: localizedName{
				Localized:       map[string]string{"en_US": "Acme Corp"},
				PreferredLocale: preferredLocale{Language: "en", Country: "US"},
			},
			Description: localizedDescription{
				Localized:       map[string]string{"en_US": "Making the best products"},
				PreferredLocale: preferredLocale{Language: "en", Country: "US"},
			},
			StaffCount: 1000,
			Locations:  []organizationLocation{{LocalizedName: "San Francisco, CA"}},
			Industries: []industryTag{{LocalizedName: "Technology"}},
			WebsiteURL: "https://acme.example.com",
			FoundedOn:  foundedOn{Year: 1999},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &getCompanyAction{conn: conn}

	params, _ := json.Marshal(getCompanyParams{OrganizationID: "12345"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_company",
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

	if data["id"] != "12345" {
		t.Errorf("expected id '12345', got %v", data["id"])
	}
	if data["name"] != "Acme Corp" {
		t.Errorf("expected name 'Acme Corp', got %v", data["name"])
	}
	if data["website_url"] != "https://acme.example.com" {
		t.Errorf("expected website_url 'https://acme.example.com', got %v", data["website_url"])
	}
}

func TestGetCompany_MissingOrganizationID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getCompanyAction{conn: conn}

	params, _ := json.Marshal(getCompanyParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_company",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing organization_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetCompany_NonNumericOrganizationID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getCompanyAction{conn: conn}

	params, _ := json.Marshal(getCompanyParams{OrganizationID: "not-numeric"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_company",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-numeric organization_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetCompany_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getCompanyAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.get_company",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestPreferredString_FallsBackWhenKeyMissing(t *testing.T) {
	t.Parallel()

	localized := map[string]string{"fr_FR": "Société Acme"}
	locale := preferredLocale{Language: "en", Country: "US"}

	result := preferredString(localized, locale)
	if result != "Société Acme" {
		t.Errorf("expected fallback value 'Société Acme', got %q", result)
	}
}

func TestPreferredString_EmptyMap(t *testing.T) {
	t.Parallel()

	result := preferredString(map[string]string{}, preferredLocale{Language: "en", Country: "US"})
	if result != "" {
		t.Errorf("expected empty string for empty map, got %q", result)
	}
}
