package make

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with a valid Make API token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_token": "test-api-token-123",
	})
}
