package docusign

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with valid access token and account ID for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "test-access-token-123",
		"account_id":   "test-account-id-456",
	})
}
