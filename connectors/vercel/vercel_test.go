package vercel

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestVercelConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "vercel" {
		t.Errorf("ID() = %q, want %q", got, "vercel")
	}
}

func TestVercelConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"vercel.list_projects",
		"vercel.list_deployments",
		"vercel.get_deployment",
		"vercel.trigger_deployment",
		"vercel.promote_deployment",
		"vercel.rollback_deployment",
		"vercel.list_env_vars",
		"vercel.set_env_var",
		"vercel.delete_env_var",
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

func TestVercelConnector_ValidateCredentials(t *testing.T) {
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
			name:    "both access_token and api_key",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "oauth_token", "api_key": "test_token"}),
			wantErr: false,
		},
		{
			name:    "missing all credentials",
			creds:   connectors.NewCredentials(map[string]string{}),
			wantErr: true,
		},
		{
			name:    "empty api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": ""}),
			wantErr: true,
		},
		{
			name:    "empty access_token",
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

func TestGetBearerToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		creds connectors.Credentials
		want  string
	}{
		{
			name:  "prefers access_token over api_key",
			creds: connectors.NewCredentials(map[string]string{"access_token": "oauth", "api_key": "key"}),
			want:  "oauth",
		},
		{
			name:  "falls back to api_key",
			creds: connectors.NewCredentials(map[string]string{"api_key": "key"}),
			want:  "key",
		},
		{
			name:  "returns empty when neither present",
			creds: connectors.NewCredentials(map[string]string{}),
			want:  "",
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

func TestVercelConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "vercel" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "vercel")
	}
	if m.Name != "Vercel" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Vercel")
	}
	if len(m.Actions) != 9 {
		t.Fatalf("Manifest().Actions has %d items, want 9", len(m.Actions))
	}
	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 2", len(m.RequiredCredentials))
	}
	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "vercel" {
		t.Errorf("oauth credential service = %q, want %q", oauthCred.Service, "vercel")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("oauth credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "vercel" {
		t.Errorf("oauth credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "vercel")
	}
	apiKeyCred := m.RequiredCredentials[1]
	if apiKeyCred.Service != "vercel-api-key" {
		t.Errorf("api_key credential service = %q, want %q", apiKeyCred.Service, "vercel-api-key")
	}
	if apiKeyCred.AuthType != "api_key" {
		t.Errorf("api_key credential auth_type = %q, want %q", apiKeyCred.AuthType, "api_key")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestVercelConnector_ActionsMatchManifest(t *testing.T) {
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

func TestVercelConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*VercelConnector)(nil)
	var _ connectors.ManifestProvider = (*VercelConnector)(nil)
}
