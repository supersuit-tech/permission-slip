package x

import (
	"context"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestXConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "x" {
		t.Errorf("ID() = %q, want %q", got, "x")
	}
}

func TestXConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"x.post_tweet",
		"x.delete_tweet",
		"x.send_dm",
		"x.get_user_tweets",
		"x.search_tweets",
		"x.get_me",
	}
	for _, key := range expected {
		if _, ok := actions[key]; !ok {
			t.Errorf("Actions() missing key %q", key)
		}
	}
	if len(actions) != len(expected) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(expected))
	}
}

func TestXConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid token",
			creds:   validCreds(),
			wantErr: false,
		},
		{
			name:    "missing token",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": ""}),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(context.Background(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if err != nil && !connectors.IsValidationError(err) {
				t.Errorf("expected ValidationError, got %T: %v", err, err)
			}
		})
	}
}

func TestXConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "x" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "x")
	}
	if m.Name != "X (Twitter)" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "X (Twitter)")
	}
	if len(m.Actions) != 6 {
		t.Errorf("Manifest().Actions has %d actions, want 6", len(m.Actions))
	}
	if len(m.Templates) != 7 {
		t.Errorf("Manifest().Templates has %d templates, want 7", len(m.Templates))
	}
	if len(m.OAuthProviders) != 1 {
		t.Errorf("Manifest().OAuthProviders has %d providers, want 1", len(m.OAuthProviders))
	} else if m.OAuthProviders[0].ID != "x" {
		t.Errorf("OAuthProviders[0].ID = %q, want %q", m.OAuthProviders[0].ID, "x")
	}
	if len(m.RequiredCredentials) != 1 {
		t.Errorf("Manifest().RequiredCredentials has %d credentials, want 1", len(m.RequiredCredentials))
	} else if m.RequiredCredentials[0].AuthType != "oauth2" {
		t.Errorf("RequiredCredentials[0].AuthType = %q, want %q", m.RequiredCredentials[0].AuthType, "oauth2")
	}

	// Validate the manifest is well-formed.
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}
