package oauth_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/oauth"
	_ "github.com/supersuit-tech/permission-slip-web/oauth/providers"
)

func TestBuiltInOAuthDisabledForKrogerAndQuickBooks(t *testing.T) {
	t.Parallel()
	for _, id := range []string{"kroger", "quickbooks"} {
		if !oauth.IsBuiltInOAuthProviderDisabled(id) {
			t.Errorf("expected %q to be disabled when connectors/%s/disabled exists", id, id)
		}
	}
}

func TestBuiltInProviders_OmitsDisabledKroger(t *testing.T) {
	t.Parallel()
	for _, p := range oauth.BuiltInProviders() {
		if p.ID == "kroger" {
			t.Fatal("kroger should not appear in BuiltInProviders when connector is disabled")
		}
	}
}
