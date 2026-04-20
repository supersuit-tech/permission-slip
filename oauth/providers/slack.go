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
			// User-token-only OAuth: use the v2_user authorize and token
			// endpoints so Slack returns the user token (xoxp-) at the top
			// level of the response. The standard v2 endpoints nest user
			// tokens inside authed_user and omit the top-level access_token
			// when no bot scopes are requested, which breaks Go's oauth2
			// library ("server response missing access_token").
			AuthorizeURL: "https://slack.com/oauth/v2_user/authorize",
			TokenURL:     "https://slack.com/api/oauth.v2.user.access",
			Scopes:       slackconnector.OAuthScopes,
			// Slack requires comma-separated scopes; the oauth2 library sends
			// space-separated. Override via AuthorizeParams so the URL is correct.
			AuthorizeParams: map[string]string{
				"scope": strings.Join(slackconnector.OAuthScopes, ","),
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
