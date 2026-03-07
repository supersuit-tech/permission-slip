package doordash

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns Credentials with valid DoorDash Drive API credentials for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"developer_id":   "test-developer-id",
		"key_id":         "test-key-id",
		"signing_secret": "test-signing-secret-at-least-32-bytes!",
	})
}
