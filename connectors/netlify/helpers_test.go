package netlify

import "github.com/supersuit-tech/permission-slip/connectors"

// validCreds returns a Credentials value with a valid API key for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "test_netlify_token",
	})
}

// oauthCreds returns a Credentials value with a valid OAuth access token for tests.
func oauthCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "test_oauth_token",
	})
}
