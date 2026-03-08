package connectors_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"

	// Blank-import connectors/providers to trigger init() registration of all
	// built-in OAuth provider IDs. See connectors/providers/doc.go for the
	// registration pattern.
	_ "github.com/supersuit-tech/permission-slip-web/connectors/providers"
)

// TestBuiltInProvidersAreRegistered verifies that the blank import of
// connectors/providers above caused all built-in provider IDs to be
// registered. If this test fails, the blank import mechanism is not working.
func TestBuiltInProvidersAreRegistered(t *testing.T) {
	ids := connectors.BuiltInOAuthProviderIDs()
	if len(ids) == 0 {
		t.Fatal("no built-in OAuth providers registered — connectors/providers blank import may not have run")
	}
}
