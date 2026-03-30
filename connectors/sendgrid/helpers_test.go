package sendgrid

import "github.com/supersuit-tech/permission-slip/connectors"

const (
	testAPIKey      = "SG.test_api_key_for_unit_tests_1234567890"
	testAccessToken = "test_oauth_access_token_for_unit_tests_1234567890"
)

// validCreds returns a Credentials value with a valid SendGrid API key for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": testAPIKey,
	})
}

// validOAuthCreds returns a Credentials value with a valid OAuth access token for tests.
func validOAuthCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": testAccessToken,
	})
}
