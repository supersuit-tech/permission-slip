// This file is split from stripe.go to keep the main connector file focused
// on struct, auth, and HTTP lifecycle. Action schemas live in manifest_actions.go
// and templates live in manifest_templates.go.
package stripe

import (
	_ "embed"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
//go:embed logo.svg
var logoSVG string

func (c *StripeConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "stripe",
		Name:        "Stripe",
		Description: "Stripe integration for payments, invoicing, billing, subscriptions, and payouts",
		LogoSVG:     logoSVG,
		Actions:     stripeActions(),
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "stripe_oauth",
				AuthType:      "oauth2",
				OAuthProvider: "stripe",
				OAuthScopes:   []string{"read_write"},
			},
			{
				Service:         "stripe",
				AuthType:        "api_key",
				InstructionsURL: "https://docs.stripe.com/keys",
			},
		},
		Templates: stripeTemplates(),
	}
}
