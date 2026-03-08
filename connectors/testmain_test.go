package connectors

import (
	"os"
	"testing"
)

// TestMain registers all built-in OAuth provider IDs before running tests.
// This mirrors what oauth/providers does via init() at runtime, but avoids
// a circular import (oauth/providers imports connectors).
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
