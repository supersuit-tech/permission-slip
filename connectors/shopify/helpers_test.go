package shopify

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with valid Shopify credentials for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"shop_domain":  "teststore",
		"access_token": "shpat_test123",
	})
}
