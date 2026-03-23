package oauth_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/oauth"
	_ "github.com/supersuit-tech/permission-slip-web/oauth/providers"
)

func TestBuiltInOAuthDisabledForKrogerQuickBooksSalesforce(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"kroger", "quickbooks", "salesforce"} {
		if !oauth.IsBuiltInOAuthProviderDisabled(id) {
			t.Errorf("expected %q to be disabled when connectors/%s/disabled exists", id, id)
		}
	}
}

func TestBuiltInProviders_OmitsDisabledConnectors(t *testing.T) {
	t.Parallel()
	for _, p := range oauth.BuiltInProviders() {
		switch p.ID {
		case "kroger", "quickbooks", "salesforce":
			t.Fatalf("%q should not appear in BuiltInProviders when connector is disabled", p.ID)
		}
	}
}
