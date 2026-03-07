package pagerduty

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with a valid API key for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "pd_test_api_key_123",
	})
}

// validOAuthCreds returns a Credentials value with a valid OAuth access token for tests.
func validOAuthCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "pd_test_oauth_token_456",
	})
}
