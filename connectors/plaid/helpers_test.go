package plaid

import "github.com/supersuit-tech/permission-slip-web/connectors"

const testClientID = "test_client_id_1234567890"
const testSecret = "test_secret_abcdefghijkl"

// validCreds returns a Credentials value with valid Plaid credentials for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"client_id": testClientID,
		"secret":    testSecret,
	})
}
