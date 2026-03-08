package connectors_test

import (
	"os"
	"testing"

	// Blank-import connectors/providers to trigger init() registration of all
	// built-in OAuth provider IDs before any test runs. This replaces the
	// previous hardcoded list in testmain_test.go. See connectors/providers/
	// doc.go for the registration pattern.
	_ "github.com/supersuit-tech/permission-slip-web/connectors/providers"
)

// TestMain sets up the test environment by ensuring all built-in OAuth
// provider IDs are registered (via the connectors/providers blank import
// above) before any test runs.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
