package connectors

import "sync"

var (
	builtInOAuthMu        sync.Mutex
	builtInOAuthProviders = make(map[string]bool)
)

// init pre-populates the built-in provider list so that ConnectorManifest.Validate
// works correctly in all contexts — including tests in connector sub-packages
// (connectors/square, connectors/asana, etc.) that do not blank-import
// connectors/providers. The connectors/providers package calls
// RegisterBuiltInOAuthProvider for the same IDs via init(), which is
// idempotent (map assignment). Production binaries blank-import
// connectors/providers too, but do not rely on it exclusively.
//
// When adding a new built-in OAuth provider, add its ID here AND create the
// corresponding files in oauth/providers/ and connectors/providers/.
func init() {
	for _, id := range []string{
		"airtable", "atlassian", "calendly", "datadog", "discord",
		"docusign", "figma", "github", "google", "hubspot", "kroger",
		"linear", "linkedin", "meta", "microsoft", "netlify", "notion",
		"pagerduty", "salesforce", "sendgrid", "shopify", "slack",
		"square", "stripe", "vercel", "zendesk", "zoom",
	} {
		builtInOAuthProviders[id] = true
	}
}

// RegisterBuiltInOAuthProvider marks an OAuth provider ID as a built-in
// platform provider. It is called from init() functions in the
// connectors/providers package. The map is already pre-populated by the
// init() above, so these calls are idempotent.
//
// Panics if id is empty to catch registration mistakes at startup rather than
// silently allowing an invalid provider into the registry.
func RegisterBuiltInOAuthProvider(id string) {
	if id == "" {
		panic("connectors.RegisterBuiltInOAuthProvider: provider ID must not be empty")
	}
	builtInOAuthMu.Lock()
	defer builtInOAuthMu.Unlock()
	builtInOAuthProviders[id] = true
}

// BuiltInOAuthProviderIDs returns all registered built-in OAuth provider IDs.
// Order is not guaranteed. Intended for testing and diagnostics.
func BuiltInOAuthProviderIDs() []string {
	builtInOAuthMu.Lock()
	defer builtInOAuthMu.Unlock()
	ids := make([]string, 0, len(builtInOAuthProviders))
	for id := range builtInOAuthProviders {
		ids = append(ids, id)
	}
	return ids
}
