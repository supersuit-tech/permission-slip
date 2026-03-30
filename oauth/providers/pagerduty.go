package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "pagerduty",
			AuthorizeURL: "https://identity.pagerduty.com/oauth/authorize",
			TokenURL:     "https://identity.pagerduty.com/oauth/token",
			Scopes: []string{
				"read",
				"write",
			},
			ClientID:     os.Getenv("PAGERDUTY_CLIENT_ID"),
			ClientSecret: os.Getenv("PAGERDUTY_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
