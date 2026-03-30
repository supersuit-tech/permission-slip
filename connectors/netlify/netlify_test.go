package netlify

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestNetlifyConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "netlify" {
		t.Errorf("ID() = %q, want %q", got, "netlify")
	}
}

func TestNetlifyConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"netlify.list_sites",
		"netlify.list_deployments",
		"netlify.get_deployment",
		"netlify.trigger_deployment",
		"netlify.rollback_deployment",
		"netlify.list_env_vars",
		"netlify.set_env_var",
		"netlify.delete_env_var",
	}
	for _, at := range want {
		if _, ok := actions[at]; !ok {
			t.Errorf("Actions() missing %q", at)
		}
	}
	if len(actions) != len(want) {
		t.Errorf("Actions() returned %d actions, want %d", len(actions), len(want))
	}
}

func TestNetlifyConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "test_token"}),
			wantErr: false,
		},
		{
			name:    "valid access_token (OAuth)",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "oauth_token"}),
			wantErr: false,
		},
		{
			name:    "both credentials present",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "key", "access_token": "token"}),
			wantErr: false,
		},
		{
			name:    "missing all credentials",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty api_key and no access_token",
			creds:   connectors.NewCredentials(map[string]string{"api_key": ""}),
			wantErr: true,
		},
		{
			name:    "empty access_token and no api_key",
			creds:   connectors.NewCredentials(map[string]string{"access_token": ""}),
			wantErr: true,
		},
		{
			name:    "zero-value credentials",
			creds:   connectors.Credentials{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil && !connectors.IsValidationError(err) {
				t.Errorf("ValidateCredentials() returned %T, want *connectors.ValidationError", err)
			}
		})
	}
}

func TestNetlifyConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "netlify" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "netlify")
	}
	if m.Name != "Netlify" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Netlify")
	}
	if len(m.Actions) != 8 {
		t.Fatalf("Manifest().Actions has %d items, want 8", len(m.Actions))
	}
	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 2", len(m.RequiredCredentials))
	}
	// First credential should be OAuth (default/primary).
	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "netlify" {
		t.Errorf("oauth credential service = %q, want %q", oauthCred.Service, "netlify")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("oauth credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "netlify" {
		t.Errorf("oauth credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "netlify")
	}
	// Second credential should be API key (alternative).
	apiKeyCred := m.RequiredCredentials[1]
	if apiKeyCred.Service != "netlify-api-key" {
		t.Errorf("api_key credential service = %q, want %q", apiKeyCred.Service, "netlify-api-key")
	}
	if apiKeyCred.AuthType != "api_key" {
		t.Errorf("api_key credential auth_type = %q, want %q", apiKeyCred.AuthType, "api_key")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestNetlifyConnector_ActionsMatchManifest(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()
	manifest := c.Manifest()

	manifestTypes := make(map[string]bool, len(manifest.Actions))
	for _, a := range manifest.Actions {
		manifestTypes[a.ActionType] = true
	}

	for actionType := range actions {
		if !manifestTypes[actionType] {
			t.Errorf("Actions() has %q but Manifest() does not", actionType)
		}
	}
	for _, a := range manifest.Actions {
		if _, ok := actions[a.ActionType]; !ok {
			t.Errorf("Manifest() has %q but Actions() does not", a.ActionType)
		}
	}
}

func TestNetlifyConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*NetlifyConnector)(nil)
	var _ connectors.ManifestProvider = (*NetlifyConnector)(nil)
}

func TestGetBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		creds connectors.Credentials
		want  string
	}{
		{
			name:  "access_token only",
			creds: connectors.NewCredentials(map[string]string{"access_token": "oauth_tok"}),
			want:  "oauth_tok",
		},
		{
			name:  "api_key only",
			creds: connectors.NewCredentials(map[string]string{"api_key": "api_tok"}),
			want:  "api_tok",
		},
		{
			name:  "both present prefers access_token",
			creds: connectors.NewCredentials(map[string]string{"access_token": "oauth_tok", "api_key": "api_tok"}),
			want:  "oauth_tok",
		},
		{
			name:  "neither present",
			creds: connectors.NewCredentials(map[string]string{}),
			want:  "",
		},
		{
			name:  "empty access_token falls back to api_key",
			creds: connectors.NewCredentials(map[string]string{"access_token": "", "api_key": "api_tok"}),
			want:  "api_tok",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBearerToken(tt.creds)
			if got != tt.want {
				t.Errorf("getBearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}
