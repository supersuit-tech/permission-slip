// Package providers registers the platform's built-in OAuth 2.0 providers.
// Each provider lives in its own file and self-registers via init(). The
// package must be blank-imported by the binary entrypoint to activate all
// registrations:
//
//	import _ "github.com/supersuit-tech/permission-slip-web/oauth/providers"
//
// # Adding a New Provider
//
// 1. Create a new file named after the provider (e.g., "acme.go").
// 2. Copy the template below and fill in the fields.
// 3. Set the ACME_CLIENT_ID and ACME_CLIENT_SECRET environment variables in
//    the server config.
// 4. That's it — no other files need editing.
//
// Template:
//
//	package providers
//
//	import (
//		"os"
//
//		"github.com/supersuit-tech/permission-slip-web/connectors"
//		"github.com/supersuit-tech/permission-slip-web/oauth"
//	)
//
//	func init() {
//		oauth.RegisterBuiltIn(oauth.Provider{
//			ID:           "acme",
//			AuthorizeURL: "https://acme.example.com/oauth/authorize",
//			TokenURL:     "https://acme.example.com/oauth/token",
//			Scopes: []string{
//				"read",
//				"write",
//			},
//			ClientID:     os.Getenv("ACME_CLIENT_ID"),
//			ClientSecret: os.Getenv("ACME_CLIENT_SECRET"),
//			Source:       oauth.SourceBuiltIn,
//		})
//		connectors.RegisterBuiltInOAuthProvider("acme")
//	}
//
// If your provider sources its OAuth scopes from the connector package (e.g.,
// because the connector manifest and the OAuth registration must stay in sync),
// import the connector package and reference its exported scope constant instead
// of duplicating the list. See slack.go, discord.go, or datadog.go for examples.
//
// If your provider needs non-standard authorize URL parameters (like Atlassian's
// audience or Slack's comma-separated scope format), add an AuthorizeParams map.
// See atlassian.go or slack.go for examples.
package providers
