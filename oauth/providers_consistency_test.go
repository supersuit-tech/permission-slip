package oauth_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/providers"
	"github.com/supersuit-tech/permission-slip-web/oauth"
	_ "github.com/supersuit-tech/permission-slip-web/oauth/providers"
)

// TestProviderRegistryConsistency verifies that every provider registered in
// oauth/providers also has a corresponding entry in connectors/providers, and
// vice versa. This catches the common mistake of adding a provider to one
// package but forgetting the other.
func TestProviderRegistryConsistency(t *testing.T) {
	oauthProviders := oauth.BuiltInProviders()
	connectorIDs := connectors.BuiltInOAuthProviderIDs()

	oauthSet := make(map[string]bool, len(oauthProviders))
	for _, p := range oauthProviders {
		oauthSet[p.ID] = true
	}

	connectorSet := make(map[string]bool, len(connectorIDs))
	for _, id := range connectorIDs {
		connectorSet[id] = true
	}

	for id := range oauthSet {
		if !connectorSet[id] {
			t.Errorf("oauth/providers registered %q but connectors/providers did not — add connectors/providers/%s.go", id, id)
		}
	}
	for id := range connectorSet {
		if !oauthSet[id] {
			t.Errorf("connectors/providers registered %q but oauth/providers did not — add oauth/providers/%s.go", id, id)
		}
	}
}
