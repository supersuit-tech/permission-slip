package docusign

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestValidateBaseURL_ValidDocuSignDomains(t *testing.T) {
	t.Parallel()

	validURLs := []string{
		"https://demo.docusign.net/restapi/v2.1",
		"https://na1.docusign.net/restapi/v2.1",
		"https://na4.docusign.net/restapi/v2.1",
		"https://eu.docusign.net/restapi/v2.1",
		"https://au.docusign.net/restapi/v2.1",
	}
	for _, u := range validURLs {
		if err := validateBaseURL(u); err != nil {
			t.Errorf("expected %q to be valid, got: %v", u, err)
		}
	}
}

func TestValidateBaseURL_RejectsNonDocuSign(t *testing.T) {
	t.Parallel()

	invalidURLs := []string{
		"https://evil.com/restapi/v2.1",
		"https://docusign.net.evil.com/restapi/v2.1",
		"http://demo.docusign.net/restapi/v2.1", // HTTP not allowed
		"https://internal-server:8080/api",
		"https://169.254.169.254/metadata", // SSRF target
	}
	for _, u := range invalidURLs {
		if err := validateBaseURL(u); err == nil {
			t.Errorf("expected %q to be rejected", u)
		}
	}
}

func TestResolveBaseURL_DefaultsToSandbox(t *testing.T) {
	t.Parallel()

	conn := New()
	creds := connectors.NewCredentials(map[string]string{
		"access_token": "tok",
		"account_id":   "acct",
	})
	url, err := conn.resolveBaseURL(creds)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != defaultBaseURL {
		t.Errorf("expected default base URL, got %q", url)
	}
}

func TestResolveBaseURL_RejectsSSRF(t *testing.T) {
	t.Parallel()

	conn := New()
	creds := connectors.NewCredentials(map[string]string{
		"access_token": "tok",
		"account_id":   "acct",
		"base_url":     "https://evil.com/steal-tokens",
	})
	_, err := conn.resolveBaseURL(creds)
	if err == nil {
		t.Fatal("expected error for non-DocuSign base_url")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestRequireAccountID_MissingReturnsError(t *testing.T) {
	t.Parallel()

	creds := connectors.NewCredentials(map[string]string{
		"access_token": "tok",
	})
	_, err := requireAccountID(creds)
	if err == nil {
		t.Fatal("expected error for missing account_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestRequireAccountID_EmptyReturnsError(t *testing.T) {
	t.Parallel()

	creds := connectors.NewCredentials(map[string]string{
		"access_token": "tok",
		"account_id":   "",
	})
	_, err := requireAccountID(creds)
	if err == nil {
		t.Fatal("expected error for empty account_id")
	}
}

func TestMissingAccountID_ReturnsValidationError(t *testing.T) {
	t.Parallel()

	// Use a real test server so the error doesn't come from the HTTP layer
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not have made an API call with missing account_id")
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &sendEnvelopeAction{conn: conn}

	params, _ := json.Marshal(sendEnvelopeParams{EnvelopeID: "env-123"})
	creds := connectors.NewCredentials(map[string]string{
		"access_token": "tok",
		// account_id deliberately missing
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "docusign.send_envelope",
		Parameters:  params,
		Credentials: creds,
	})
	if err == nil {
		t.Fatal("expected error for missing account_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
