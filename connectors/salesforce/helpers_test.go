package salesforce

import "github.com/supersuit-tech/permission-slip/connectors"

// validCreds returns a Credentials value with a valid access token and
// instance URL for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "00Dxx0000000000!test-access-token-123",
		"instance_url": "https://myorg.salesforce.com",
	})
}
