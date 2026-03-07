package datadog

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with valid API and app keys for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "dd_test_api_key_123",
		"app_key": "dd_test_app_key_456",
	})
}
