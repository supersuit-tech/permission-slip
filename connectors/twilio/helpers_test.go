package twilio

import "github.com/supersuit-tech/permission-slip-web/connectors"

const testAccountSID = "AC12345678901234567890123456789012"
const testAuthToken = "test_auth_token_abc123"

// validCreds returns a Credentials value with valid Twilio credentials for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"account_sid": testAccountSID,
		"auth_token":  testAuthToken,
	})
}
