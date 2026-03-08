package connectors

import (
	"os"
	"testing"
)

// TestMain registers all built-in OAuth provider IDs before running tests.
// This ensures tests that call ParseManifest (which validates oauth_provider
// references against builtInOAuthProviders) find the expected providers.
//
// The connectors/testsetup_test.go file also blank-imports
// connectors/providers as the init()-based registration mechanism, but this
// explicit fallback in TestMain guarantees providers are registered even if
// the external test package's init() does not run.
func TestMain(m *testing.M) {
	for _, id := range []string{
		"airtable", "atlassian", "calendly", "datadog", "discord",
		"docusign", "figma", "github", "google", "hubspot", "kroger",
		"linear", "linkedin", "meta", "microsoft", "netlify", "notion",
		"pagerduty", "salesforce", "sendgrid", "shopify", "slack",
		"square", "stripe", "vercel", "zendesk", "zoom",
	} {
		RegisterBuiltInOAuthProvider(id)
	}
	os.Exit(m.Run())
}
