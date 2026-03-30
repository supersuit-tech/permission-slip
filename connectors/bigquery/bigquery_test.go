package bigquery

import (
	"encoding/json"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func minimalServiceAccountJSON(project string) string {
	b, _ := json.Marshal(map[string]string{
		"type":                        "service_account",
		"project_id":                  project,
		"private_key_id":              "x",
		"private_key":                 "-----BEGIN PRIVATE KEY-----\nMIIE\n-----END PRIVATE KEY-----\n",
		"client_email":                "x@x.iam.gserviceaccount.com",
		"client_id":                   "1",
		"auth_uri":                    "https://accounts.google.com/o/oauth2/auth",
		"token_uri":                   "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
	})
	return string(b)
}

func TestBigQueryConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "bigquery" {
		t.Errorf("ID() = %q", c.ID())
	}
}

func TestBigQueryConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(t.Context(), connectors.NewCredentials(map[string]string{
		"service_account_json": minimalServiceAccountJSON("myproj"),
	}))
	if err != nil {
		t.Errorf("unexpected: %v", err)
	}
	err = c.ValidateCredentials(t.Context(), connectors.NewCredentials(map[string]string{}))
	if err == nil || !connectors.IsValidationError(err) {
		t.Fatalf("expected validation error, got %v", err)
	}
}
