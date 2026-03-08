package connectors

import (
	"os"
	"testing"
)

// TestMain is the test entry point. Built-in OAuth provider IDs are already
// pre-populated by the init() in builtin_oauth_providers.go, so no explicit
// registration is needed here.
func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
