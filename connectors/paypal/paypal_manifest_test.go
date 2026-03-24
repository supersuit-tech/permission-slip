package paypal

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

var _ connectors.Connector = (*PayPalConnector)(nil)
var _ connectors.ManifestProvider = (*PayPalConnector)(nil)

func TestManifest_Valid(t *testing.T) {
	t.Parallel()

	conn := New()
	m := conn.Manifest()
	if err := m.Validate(); err != nil {
		t.Fatalf("Manifest().Validate(): %v", err)
	}
	if m.ID != "paypal" {
		t.Errorf("ID = %q, want paypal", m.ID)
	}
	if len(m.Actions) != 11 {
		t.Errorf("Actions count = %d, want 11", len(m.Actions))
	}
	if len(m.RequiredCredentials) != 1 || m.RequiredCredentials[0].OAuthProvider != "paypal" {
		t.Errorf("credentials: %+v", m.RequiredCredentials)
	}
}
