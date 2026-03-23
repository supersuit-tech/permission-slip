package oauth_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/oauth"
	_ "github.com/supersuit-tech/permission-slip-web/oauth/providers"
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
	disabled := map[string]bool{"kroger": true}
	for _, p := range oauth.BuiltInProviders() {
		if disabled[p.ID] {
			t.Errorf("%q should not appear in BuiltInProviders when connector is disabled", p.ID)
		}
	}
}
