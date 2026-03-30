package slack

import "github.com/supersuit-tech/permission-slip/connectors"

// validCreds returns a Credentials value with a valid user OAuth token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "xoxp-test-token-123",
	})
}
