package github

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestGitHubConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "github" {
		t.Errorf("ID() = %q, want %q", got, "github")
	}
}

func TestGitHubConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"github.create_issue",
		"github.merge_pr",
		"github.create_pr",
		"github.add_reviewer",
		"github.create_release",
		"github.close_issue",
		"github.add_label",
		"github.add_comment",
		"github.create_branch",
		"github.get_file_contents",
		"github.create_or_update_file",
		"github.list_repos",
		"github.get_repo",
		"github.list_pull_requests",
		"github.trigger_workflow",
		"github.list_workflow_runs",
		"github.create_webhook",
		"github.search_code",
		"github.search_issues",
		"github.create_repo",
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

func TestGitHubConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{
			name:    "valid api_key",
			creds:   connectors.NewCredentials(map[string]string{"api_key": "ghp_test123"}),
			wantErr: false,
		},
		{
			name:    "valid access_token (OAuth)",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "gho_test123"}),
			wantErr: false,
		},
		{
			name:    "both present prefers access_token",
			creds:   connectors.NewCredentials(map[string]string{"access_token": "gho_test", "api_key": "ghp_test"}),
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
			name:    "wrong key name",
			creds:   connectors.NewCredentials(map[string]string{"token": "ghp_test123"}),
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

func TestGitHubConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "github" {
		t.Errorf("Manifest().ID = %q, want %q", m.ID, "github")
	}
	if m.Name != "GitHub" {
		t.Errorf("Manifest().Name = %q, want %q", m.Name, "GitHub")
	}
	if len(m.Actions) != 20 {
		t.Fatalf("Manifest().Actions has %d items, want 20", len(m.Actions))
	}
	actionTypes := make(map[string]bool)
	for _, a := range m.Actions {
		actionTypes[a.ActionType] = true
	}
	for _, want := range []string{
		"github.create_issue", "github.merge_pr", "github.create_pr",
		"github.add_reviewer", "github.create_release", "github.close_issue",
		"github.add_label", "github.add_comment", "github.create_branch",
		"github.get_file_contents", "github.create_or_update_file",
		"github.list_repos", "github.get_repo", "github.list_pull_requests",
		"github.trigger_workflow", "github.list_workflow_runs",
		"github.create_webhook", "github.search_code", "github.search_issues",
		"github.create_repo",
	} {
		if !actionTypes[want] {
			t.Errorf("Manifest().Actions missing %q", want)
		}
	}
	if len(m.RequiredCredentials) != 2 {
		t.Fatalf("Manifest().RequiredCredentials has %d items, want 2", len(m.RequiredCredentials))
	}
	// First credential should be OAuth (primary).
	oauthCred := m.RequiredCredentials[0]
	if oauthCred.Service != "github" {
		t.Errorf("oauth credential service = %q, want %q", oauthCred.Service, "github")
	}
	if oauthCred.AuthType != "oauth2" {
		t.Errorf("oauth credential auth_type = %q, want %q", oauthCred.AuthType, "oauth2")
	}
	if oauthCred.OAuthProvider != "github" {
		t.Errorf("oauth credential oauth_provider = %q, want %q", oauthCred.OAuthProvider, "github")
	}
	if len(oauthCred.OAuthScopes) == 0 {
		t.Error("oauth credential oauth_scopes is empty, want at least one scope")
	}
	// Second credential should be API key (alternative).
	patCred := m.RequiredCredentials[1]
	if patCred.Service != "github_pat" {
		t.Errorf("pat credential service = %q, want %q", patCred.Service, "github_pat")
	}
	if patCred.AuthType != "api_key" {
		t.Errorf("pat credential auth_type = %q, want %q", patCred.AuthType, "api_key")
	}
	if patCred.InstructionsURL == "" {
		t.Error("pat credential instructions_url is empty, want a URL")
	}

	// Validate the manifest passes validation.
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestGitHubConnector_ActionsMatchManifest(t *testing.T) {
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

func TestGitHubConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*GitHubConnector)(nil)
	var _ connectors.ManifestProvider = (*GitHubConnector)(nil)
}
