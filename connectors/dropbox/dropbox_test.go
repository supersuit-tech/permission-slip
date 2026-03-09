package dropbox

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestDropboxConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if c.ID() != "dropbox" {
		t.Errorf("expected ID 'dropbox', got %q", c.ID())
	}
}

func TestDropboxConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	expected := []string{
		"dropbox.upload_file",
		"dropbox.download_file",
		"dropbox.create_folder",
		"dropbox.share_file",
		"dropbox.search",
		"dropbox.move",
	}
	for _, name := range expected {
		if _, ok := actions[name]; !ok {
			t.Errorf("expected action %q to be registered", name)
		}
	}

	if len(actions) != len(expected) {
		t.Errorf("expected %d actions, got %d", len(expected), len(actions))
	}
}

func TestDropboxConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "sl.test-token"}),
			wantErr: false,
		},
		{
			name:    "missing access_token",
			creds:   connectors.NewCredentials(map[string]string{}),
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

func TestDropboxConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "dropbox" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "dropbox")
	}
	if m.Name != "Dropbox" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "Dropbox")
	}
	if len(m.Actions) != 6 {
		t.Fatalf("Manifest().Actions has %d items, want 6", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"dropbox.upload_file", "dropbox.download_file", "dropbox.create_folder",
		"dropbox.share_file", "dropbox.search", "dropbox.move",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 1 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 1", len(m.RequiredCredentials))
	}

	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "dropbox" {
		t.Errorf("oauth credential service = %q, want %q", oauthCred.Service, "dropbox")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("oauth credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "dropbox" {
		t.Errorf("oauth credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "dropbox")
	}
	if len(oauthCred.OAuthScopes) == 0 {
		t.Error("oauth credential oauth_scopes is empty, want at least one scope")
	}

	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestDropboxConnector_ActionsMatchManifest(t *testing.T) {
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

func TestDropboxConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*DropboxConnector)(nil)
	var _ connectors.ManifestProvider = (*DropboxConnector)(nil)
}

func TestValidatePath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "valid absolute path", path: "/Documents/report.pdf", wantErr: false},
		{name: "valid root path", path: "/test", wantErr: false},
		{name: "empty", path: "", wantErr: true},
		{name: "relative path", path: "Documents/report.pdf", wantErr: true},
		{name: "trailing slash", path: "/Documents/", wantErr: true},
		{name: "double slashes", path: "/Documents//report.pdf", wantErr: true},
		{name: "root slash only", path: "/", wantErr: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := validatePath(tt.path, "path")
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
