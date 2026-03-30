package expedia

import "github.com/supersuit-tech/permission-slip/connectors"

// validCreds returns a Credentials value with valid api_key and secret for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "test_api_key",
		"secret":  "test_secret",
	})
}
