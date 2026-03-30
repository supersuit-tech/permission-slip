package zendesk

import "github.com/supersuit-tech/permission-slip/connectors"

// validCreds returns a Credentials value with valid Zendesk credentials for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"subdomain": "testcompany",
		"email":     "agent@example.com",
		"api_token": "test-api-token-123",
	})
}
