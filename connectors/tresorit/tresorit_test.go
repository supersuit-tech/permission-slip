package tresorit

import (
	"context"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestTresoritConnector_ID(t *testing.T) {
	t.Parallel()
	c := New()
	if got := c.ID(); got != "tresorit" {
		t.Errorf("ID() = %q, want %q", got, "tresorit")
	}
}

func TestTresoritConnector_Actions(t *testing.T) {
	t.Parallel()
	c := New()
	actions := c.Actions()

	want := []string{
		"tresorit.list_files",
		"tresorit.download_file",
		"tresorit.upload_file",
		"tresorit.create_folder",
		"tresorit.delete_file",
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

func TestTresoritConnector_ValidateCredentials(t *testing.T) {
	t.Parallel()
	c := New()

	old := testListBuckets
	testListBuckets = func(_ context.Context, _ *TresoritConnector, _ connectors.Credentials, _ time.Duration) error {
		return nil
	}
	t.Cleanup(func() { testListBuckets = old })

	tests := []struct {
		name    string
		creds   connectors.Credentials
		wantErr bool
	}{
		{name: "valid credentials", creds: validCreds(), wantErr: false},
		{name: "missing access_key", creds: connectors.NewCredentials(map[string]string{
			credKeySecretKey: "secret", credKeyEndpointURL: "http://127.0.0.1:3000",
		}), wantErr: true},
		{name: "missing secret_key", creds: connectors.NewCredentials(map[string]string{
			credKeyAccessKey: "key", credKeyEndpointURL: "http://127.0.0.1:3000",
		}), wantErr: true},
		{name: "missing endpoint_url", creds: connectors.NewCredentials(map[string]string{
			credKeyAccessKey: "key", credKeySecretKey: "secret",
		}), wantErr: true},
		{name: "invalid endpoint_url", creds: connectors.NewCredentials(map[string]string{
			credKeyAccessKey: "key", credKeySecretKey: "secret", credKeyEndpointURL: "not-a-url",
		}), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := c.ValidateCredentials(t.Context(), tt.creds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCredentials() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTresoritConnector_Manifest(t *testing.T) {
	t.Parallel()
	c := New()
	m := c.Manifest()

	if m.ID != "tresorit" {
		t.Errorf("Manifest().ID = %q, want tresorit", m.ID)
	}
	if len(m.Actions) != 5 {
		t.Fatalf("Manifest().Actions has %d items, want 5", len(m.Actions))
	}
	if err := m.Validate(); err != nil {
		t.Errorf("Manifest().Validate() = %v", err)
	}
}

func TestTresoritConnector_ActionsMatchManifest(t *testing.T) {
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

func TestTresoritConnector_ImplementsInterface(t *testing.T) {
	t.Parallel()
	var _ connectors.Connector = (*TresoritConnector)(nil)
	var _ connectors.ManifestProvider = (*TresoritConnector)(nil)
}

func TestObjectPath(t *testing.T) {
	t.Parallel()
	tests := []struct {
		tresor, key, want string
	}{
		{"my-tresor", "", "/my-tresor"},
		{"my-tresor", "docs/report.pdf", "/my-tresor/docs/report.pdf"},
		{"my-tresor", "/docs/report.pdf", "/my-tresor/docs/report.pdf"},
	}
	for _, tt := range tests {
		if got := objectPath(tt.tresor, tt.key); got != tt.want {
			t.Errorf("objectPath(%q, %q) = %q, want %q", tt.tresor, tt.key, got, tt.want)
		}
	}
}
