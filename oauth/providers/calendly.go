package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(oauth.Provider{
		ID:           "calendly",
		AuthorizeURL: "https://auth.calendly.com/oauth/authorize",
		TokenURL:     "https://auth.calendly.com/oauth/token",
		Scopes:       []string{},
		ClientID:     os.Getenv("CALENDLY_CLIENT_ID"),
		ClientSecret: os.Getenv("CALENDLY_CLIENT_SECRET"),
		Source:       oauth.SourceBuiltIn,
	})
	connectors.RegisterBuiltInOAuthProvider("calendly")
}
