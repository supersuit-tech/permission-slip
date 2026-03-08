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
			// standard space-separated format.
			AuthorizeParams: map[string]string{
				"scope": strings.Join(slackconnector.OAuthScopes, ","),
			},
			ClientID:     os.Getenv("SLACK_CLIENT_ID"),
			ClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
