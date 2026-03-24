package paypal

import (
	_ "embed"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

//go:embed logo.svg
var logoSVG string

func (c *PayPalConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "paypal",
		Name:        "PayPal / Venmo",
		Description: "PayPal REST API for Venmo payouts, Checkout orders (including Venmo funding), invoicing, and refunds. Uses OAuth 2.0; set credential environment to sandbox for api-m.sandbox.paypal.com.",
		LogoSVG:     logoSVG,
		Actions:     paypalActions(),
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "paypal",
				AuthType:        "oauth2",
				OAuthProvider:   "paypal",
				OAuthScopes:     OAuthScopes,
				InstructionsURL: "https://developer.paypal.com/dashboard/",
			},
		},
	}
}
