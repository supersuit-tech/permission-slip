package slack

import (
	"fmt"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// Slack token rotation can wrap the token type in an xoxe prefix. Match on the
// embedded token family so legacy bot tokens (xoxb-) are rejected for actions
// that must act as the authorizing user.
func looksLikeSlackUserToken(token string) bool {
	return strings.HasPrefix(token, "xoxp-") || strings.Contains(token, "xoxp-")
}

func looksLikeSlackBotToken(token string) bool {
	return strings.HasPrefix(token, "xoxb-") || strings.Contains(token, "xoxb-")
}

func (c *SlackConnector) requireUserOAuthToken(creds connectors.Credentials, action string) error {
	token, err := c.getToken(creds)
	if err != nil {
		return err
	}
	if looksLikeSlackBotToken(token) && !looksLikeSlackUserToken(token) {
		return &connectors.AuthError{
			Message: fmt.Sprintf("%s requires a Slack user OAuth token (xoxp-), but this connection is still using a legacy bot token — reconnect Slack", action),
		}
	}
	return nil
}
