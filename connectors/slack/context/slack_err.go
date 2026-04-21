package context

import (
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func mapSlackErr(code string) error {
	switch code {
	case "not_authed", "invalid_auth", "token_revoked", "token_expired", "account_inactive":
		return &connectors.AuthError{Message: fmt.Sprintf("Slack auth error: %s", code)}
	case "missing_scope":
		return &connectors.AuthError{Message: "Slack token is missing a required OAuth scope — re-authorize the Slack connection"}
	case "ratelimited":
		return &connectors.RateLimitError{Message: "Slack API rate limit exceeded"}
	default:
		return &connectors.ExternalError{StatusCode: 200, Message: fmt.Sprintf("Slack API error: %s", code)}
	}
}
