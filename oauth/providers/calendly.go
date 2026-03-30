package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "calendly",
			AuthorizeURL: "https://auth.calendly.com/oauth/authorize",
			TokenURL:     "https://auth.calendly.com/oauth/token",
			Scopes:       []string{},
			ClientID:     os.Getenv("CALENDLY_CLIENT_ID"),
			ClientSecret: os.Getenv("CALENDLY_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
