package quickbooks

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with valid test credentials.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "test-access-token-abc123",
		"realm_id":     "1234567890",
	})
}
