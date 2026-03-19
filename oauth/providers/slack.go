package providers

import (
	"os"
	"strings"

	slackconnector "github.com/supersuit-tech/permission-slip-web/connectors/slack"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "slack",
			AuthorizeURL: "https://slack.com/oauth/v2/authorize",
			TokenURL:     "https://slack.com/api/oauth.v2.access",
			Scopes:       slackconnector.OAuthScopes,
			// Slack V2 OAuth requires comma-separated scopes instead of the
			// standard space-separated format. User scopes (for endpoints
			// like search.messages that require a user token) are passed via
			// the separate "user_scope" parameter.
			AuthorizeParams: map[string]string{
				"scope":      strings.Join(slackconnector.OAuthScopes, ","),
				"user_scope": strings.Join(slackconnector.OAuthUserScopes, ","),
			},
			ClientID:     os.Getenv("SLACK_CLIENT_ID"),
			ClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
