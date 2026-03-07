package pagerduty

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with a valid API key for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "pd_test_api_key_123",
	})
}
