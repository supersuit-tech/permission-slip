package trello

import "github.com/supersuit-tech/permission-slip-web/connectors"

// Test IDs that pass Trello ID validation (24-char hex strings).
const (
	testCardID      = "507f1f77bcf86cd799439011"
	testListID      = "507f1f77bcf86cd799439022"
	testBoardID     = "507f1f77bcf86cd799439033"
	testChecklistID = "507f1f77bcf86cd799439044"
)

// validCreds returns a Credentials value with valid api_key and token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"api_key": "test-api-key-123",
		"token":   "test-token-456",
	})
}

// oauthCreds returns a Credentials value with a valid OAuth access_token for tests.
func oauthCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"access_token": "test-oauth-access-token-789",
	})
}
