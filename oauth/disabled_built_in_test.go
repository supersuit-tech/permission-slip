package oauth_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
	"github.com/supersuit-tech/permission-slip/oauth"
	_ "github.com/supersuit-tech/permission-slip/oauth/providers"
)

func TestBuiltInOAuthDisabledForKrogerNotQuickBooks(t *testing.T) {
	t.Parallel()
	if !oauth.IsBuiltInOAuthProviderDisabled("kroger") {
		t.Errorf("expected %q to be disabled when connectors/%s/disabled exists", "kroger", "kroger")
	}
	if oauth.IsBuiltInOAuthProviderDisabled("quickbooks") {
		t.Errorf("expected %q not to be disabled (connector re-enabled)", "quickbooks")
	}
}

func TestBuiltInProviders_OmitsDisabledConnectors(t *testing.T) {
	t.Parallel()
	disabled := make(map[string]bool)
	for _, id := range connectors.DisabledBuiltInConnectorIDs() {
		disabled[id] = true
	}
	if len(disabled) == 0 {
		t.Skip("no connectors currently disabled; nothing to verify")
	}
	for _, p := range oauth.BuiltInProviders() {
		if disabled[p.ID] {
			t.Errorf("%q should not appear in BuiltInProviders when connector is disabled", p.ID)
		}
	}
}
