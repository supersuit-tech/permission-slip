package meta

import "github.com/supersuit-tech/permission-slip/connectors"

// validCreds returns a Credentials value with a valid access token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "EAAtest-access-token-123",
	})
}
