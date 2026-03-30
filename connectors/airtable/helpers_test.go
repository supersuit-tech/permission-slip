package airtable

import "github.com/supersuit-tech/permission-slip/connectors"

// validCreds returns a Credentials value with a valid personal access token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_token": "patTEST1234567890.abcdef",
	})
}
