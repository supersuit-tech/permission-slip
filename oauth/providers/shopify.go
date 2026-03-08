package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			// Shopify uses per-shop OAuth URLs. The {shop} placeholder is
			// replaced at authorize/callback time with the user's shop
			// subdomain (e.g. "mystore"). See api/oauth.go for resolution.
			ID:           "shopify",
			AuthorizeURL: "https://{shop}.myshopify.com/admin/oauth/authorize",
			TokenURL:     "https://{shop}.myshopify.com/admin/oauth/access_token",
			Scopes: []string{
				"write_orders",
				"write_products",
				"write_inventory",
				"write_discounts",
				"read_reports",
				"read_all_orders",
			},
			ClientID:     os.Getenv("SHOPIFY_CLIENT_ID"),
			ClientSecret: os.Getenv("SHOPIFY_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
