package intercom

import "github.com/supersuit-tech/permission-slip/connectors"

// validCreds returns a Credentials value with a valid Intercom access token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "dG9rOmFiY2RlZmcxMjM0NTY3ODk=",
	})
}
