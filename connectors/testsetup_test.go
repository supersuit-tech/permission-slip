package connectors_test

import (
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"

	// Blank-import connectors/all to trigger init() self-registration of all
	// built-in connectors. This also transitively imports connectors/providers.
	_ "github.com/supersuit-tech/permission-slip-web/connectors/all"
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
	for _, disabledID := range connectors.DisabledBuiltInConnectorIDs() {
		for _, id := range ids {
			if id == disabledID {
				t.Errorf("BuiltInOAuthProviderIDs: disabled connector %q should not appear", disabledID)
			}
		}
	}
}

// TestBuiltInConnectorsAreRegistered verifies that importing connectors/all
// triggers init() registration for all built-in connectors. If a new
// connector package is added but its register.go or blank import in all.go
// is missing, the count will drop and this test will fail.
func TestBuiltInConnectorsAreRegistered(t *testing.T) {
	got := connectors.BuiltInConnectors()
	// There are 54 active built-in connectors; kroger is disabled via
	// connectors/kroger/disabled (embedded in the binary via //go:embed).
	// Update this number when adding, removing, or re-enabling connectors.
	const expected = 54
	if len(got) != expected {
		t.Fatalf("expected %d built-in connectors, got %d — did you forget to add register.go or a blank import in connectors/all?", expected, len(got))
	}

	// Verify no duplicate IDs slipped through.
	seen := make(map[string]bool, len(got))
	for _, c := range got {
		if seen[c.ID()] {
			t.Errorf("duplicate built-in connector ID: %q", c.ID())
		}
		seen[c.ID()] = true
	}
}
