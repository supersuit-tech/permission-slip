package docusign

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "docusign" {
		t.Errorf("expected ID docusign, got %q", got)
	}
}

func TestManifest_Valid(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()
	if err := m.Validate(); err != nil {
		t.Fatalf("manifest validation failed: %v", err)
	}
	if m.ID != "docusign" {
		t.Errorf("expected manifest ID docusign, got %q", m.ID)
	}
	if len(m.Actions) != 7 {
		t.Errorf("expected 7 actions, got %d", len(m.Actions))
	}
	if len(m.Templates) != 7 {
		t.Errorf("expected 7 templates, got %d", len(m.Templates))
	}
}

func TestManifest_Credentials(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("expected 2 required credentials, got %d", len(m.RequiredCredentials))
	}

	// First credential: OAuth2 (primary / recommended)
	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "docusign" {
		t.Errorf("oauth credential service = %q, want %q", oauthCred.Service, "docusign")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("oauth credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "docusign" {
		t.Errorf("oauth credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "docusign")
	}
	if len(oauthCred.OAuthScopes) == 0 {
		t.Error("oauth credential oauth_scopes is empty, want at least one scope")
	}

	// Second credential: custom / RSA key auth (alternative)
	customCred := m.RequiredCredentials[1]
	if customCred.Service != "docusign" {
		t.Errorf("custom credential service = %q, want %q", customCred.Service, "docusign")
	}
	if customCred.AuthType != "custom" {
		t.Errorf("custom credential auth_type = %q, want %q", customCred.AuthType, "custom")
	}
	if customCred.InstructionsURL == "" {
		t.Error("custom credential instructions_url is empty, want a URL")
	}
}

func TestValidateCredentials_Valid(t *testing.T) {
	t.Parallel()
	c := New()
	err := c.ValidateCredentials(t.Context(), validCreds())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateCredentials_MissingAccessToken(t *testing.T) {
	t.Parallel()
	c := New()
	creds := connectors.NewCredentials(map[string]string{
		"account_id": "test-account",
	})
	err := c.ValidateCredentials(t.Context(), creds)
	if err == nil {
		t.Fatal("expected error for missing access_token")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestValidateCredentials_MissingAccountID(t *testing.T) {
	t.Parallel()
	c := New()
	creds := connectors.NewCredentials(map[string]string{
		"access_token": "test-token",
	})
	err := c.ValidateCredentials(t.Context(), creds)
	if err == nil {
		t.Fatal("expected error for missing account_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestActions_AllRegistered(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"docusign.create_envelope",
		"docusign.send_envelope",
		"docusign.check_status",
		"docusign.download_signed",
		"docusign.list_templates",
		"docusign.void_envelope",
		"docusign.update_recipients",
	}

	for _, actionType := range expected {
		if _, ok := actions[actionType]; !ok {
			t.Errorf("expected action %q to be registered", actionType)
		}
	}

	if len(actions) != len(expected) {
		t.Errorf("expected %d actions, got %d", len(expected), len(actions))
	}
}
