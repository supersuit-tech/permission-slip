package cloudflare

import "github.com/supersuit-tech/permission-slip-web/connectors"

func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "test-api-token-123",
	})
}
