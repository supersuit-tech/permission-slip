package doordash

import (
	"encoding/base64"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validCreds returns Credentials with valid DoorDash Drive API credentials for tests.
// The signing_secret is base64url-encoded, matching the format DoorDash provides.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"developer_id":   "test-developer-id",
		"key_id":         "test-key-id",
		"signing_secret": base64.RawURLEncoding.EncodeToString([]byte("test-signing-secret-at-least-32-bytes!")),
	})
}
