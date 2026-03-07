package jira

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with valid Jira credentials for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"site":      "testsite",
		"email":     "user@example.com",
		"api_token": "test-api-token-123",
	})
}
