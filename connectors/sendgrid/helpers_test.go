package sendgrid

import "github.com/supersuit-tech/permission-slip-web/connectors"

const testAPIKey = "SG.test_api_key_for_unit_tests_1234567890"

// validCreds returns a Credentials value with a valid SendGrid API key for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": testAPIKey,
	})
}
