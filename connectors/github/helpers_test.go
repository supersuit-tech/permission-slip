package github

import "github.com/supersuit-tech/permission-slip/connectors"

// validCreds returns a Credentials value with a valid API key for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "ghp_test123",
	})
}
