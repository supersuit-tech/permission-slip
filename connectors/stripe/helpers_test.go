package stripe

import "github.com/supersuit-tech/permission-slip/connectors"

// validCreds returns a Credentials value with a valid test API key.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "sk_test_abc123",
	})
}
