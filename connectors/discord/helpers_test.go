package discord

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with a valid bot token for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"bot_token": "test-bot-token-123",
	})
}
