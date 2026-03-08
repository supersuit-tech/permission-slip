package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "square",
			AuthorizeURL: "https://connect.squareup.com/oauth2/authorize",
			TokenURL:     "https://connect.squareup.com/oauth2/token",
			Scopes: []string{
				"ORDERS_READ",
				"ORDERS_WRITE",
				"PAYMENTS_READ",
				"PAYMENTS_WRITE",
				"ITEMS_READ",
				"ITEMS_WRITE",
				"CUSTOMERS_READ",
				"CUSTOMERS_WRITE",
				"APPOINTMENTS_READ",
				"APPOINTMENTS_WRITE",
				"INVOICES_READ",
				"INVOICES_WRITE",
				"INVENTORY_READ",
				"INVENTORY_WRITE",
			},
			ClientID:     os.Getenv("SQUARE_CLIENT_ID"),
			ClientSecret: os.Getenv("SQUARE_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
