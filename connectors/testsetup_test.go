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
}

// TestBuiltInConnectorsAreRegistered verifies that importing connectors/all
// triggers init() registration for all built-in connectors. If a new
// connector package is added but its register.go or blank import in all.go
// is missing, the count will drop and this test will fail.
func TestBuiltInConnectorsAreRegistered(t *testing.T) {
	got := connectors.BuiltInConnectors()
	// There are 50 built-in connector packages. Update this number when
	// adding or removing connectors.
	const expected = 50
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

	// ProtonMail must be in BuiltInConnectors unconditionally — the env-var
	// gating (ENABLE_PROTONMAIL_CONNECTOR) is applied in main.go, not in
	// the connector's init(). This ensures .env values loaded by godotenv
	// are visible to the check.
	if !seen["protonmail"] {
		t.Error("protonmail not found in BuiltInConnectors — it must register unconditionally; env-var gating belongs in main.go")
	}
}
