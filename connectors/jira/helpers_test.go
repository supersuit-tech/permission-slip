package jira

import "github.com/supersuit-tech/permission-slip/connectors"

// validCreds returns a Credentials value with valid basic auth Jira credentials for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"site":      "testsite",
		"email":     "user@example.com",
		"api_token": "test-api-token-123",
	})
}

// validOAuthCreds returns a Credentials value with a valid OAuth access token for tests.
func validOAuthCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "test-oauth-token-456",
	})
}
