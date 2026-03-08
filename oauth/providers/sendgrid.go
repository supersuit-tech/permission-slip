package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(oauth.Provider{
		// SendGrid is owned by Twilio and uses Twilio's OAuth 2.0
		// authorization server. The resulting access token is used as a
		// Bearer token on SendGrid v3 API requests, the same as an API key.
		ID:           "sendgrid",
		AuthorizeURL: "https://login.twilio.com/oauth2/authorize",
		TokenURL:     "https://login.twilio.com/oauth2/token",
		Scopes: []string{
			"openid",
			"profile",
			"email",
		},
		ClientID:     os.Getenv("SENDGRID_CLIENT_ID"),
		ClientSecret: os.Getenv("SENDGRID_CLIENT_SECRET"),
		Source:       oauth.SourceBuiltIn,
	})
	connectors.RegisterBuiltInOAuthProvider("sendgrid")
}
