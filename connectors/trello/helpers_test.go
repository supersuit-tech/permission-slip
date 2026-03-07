package trello

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with valid api_key and token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "test-api-key-123",
		"token":   "test-token-456",
	})
}
