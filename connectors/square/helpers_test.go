package square

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with a valid access token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "EAAAEBcXzPsWbM0yJjRvxlT_test",
	})
}

// sandboxCreds returns Credentials configured for the sandbox environment.
func sandboxCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "EAAAEBcXzPsWbM0yJjRvxlT_test",
		"environment":  "sandbox",
	})
}
