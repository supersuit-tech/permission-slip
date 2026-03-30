package providers

import (
	"os"
	"strings"

	"golang.org/x/oauth2"

	slackconnector "github.com/supersuit-tech/permission-slip/connectors/slack"
	"github.com/supersuit-tech/permission-slip/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID: "slack",
			// Standard Slack OAuth v2 endpoints. The authorize URL uses the
			// standard /oauth/v2/authorize path with user_scope (not scope)
			// to request user-level permissions. The token URL is the standard
			// oauth.v2.access endpoint. When only user scopes are requested
			// (no bot scopes), Slack nests the user token inside authed_user
			// rather than at the top level — the OAuth callback handles this
			// via NormalizeSlackUserOAuthToken.
			AuthorizeURL: "https://slack.com/oauth/v2/authorize",
			TokenURL:     "https://slack.com/api/oauth.v2.access",
			Scopes:       slackconnector.OAuthScopes,
			// Slack requires user_scope (not scope) for user-level OAuth
			// permissions, and scopes must be comma-separated (the oauth2
			// library sends space-separated). Override via AuthorizeParams.
			AuthorizeParams: map[string]string{
				"user_scope": strings.Join(slackconnector.OAuthScopes, ","),
			},
			// Slack's token endpoint requires client credentials as POST body
			// params and returns HTTP 200 for all responses (even errors).
			// AuthStyleAutoDetect tries Basic auth first and only falls back
			// on HTTP 401, so we must set AuthStyleInParams explicitly.
			AuthStyle:    oauth2.AuthStyleInParams,
			ClientID:     os.Getenv("SLACK_CLIENT_ID"),
			ClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
