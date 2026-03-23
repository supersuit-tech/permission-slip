package providers

import (
	"os"
	"strings"

	"golang.org/x/oauth2"

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
			// Slack V2 OAuth requires comma-separated scopes. User-token-only
			// apps request scopes via "user_scope" only (no bot "scope" param).
			AuthorizeParams: map[string]string{
				"user_scope": strings.Join(slackconnector.OAuthScopes, ","),
			},
			// Slack's token endpoint requires client credentials as POST body
			// params and returns HTTP 200 for all responses (even errors).
			// AuthStyleAutoDetect never retries because it only falls back on
			// HTTP 401, so we must set AuthStyleInParams explicitly.
			AuthStyle:    oauth2.AuthStyleInParams,
			ClientID:     os.Getenv("SLACK_CLIENT_ID"),
			ClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
