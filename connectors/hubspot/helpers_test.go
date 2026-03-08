package hubspot

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with a valid API key for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "pat-na1-test-token-123",
	})
}

// validOAuthCreds returns a Credentials value with a valid OAuth access token for tests.
func validOAuthCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "oauth-test-token-456",
	})
}
