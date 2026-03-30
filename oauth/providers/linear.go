package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "linear",
			AuthorizeURL: "https://linear.app/oauth/authorize",
			TokenURL:     "https://api.linear.app/oauth/token",
			Scopes: []string{
				"read",
				"write",
			},
			ClientID:     os.Getenv("LINEAR_CLIENT_ID"),
			ClientSecret: os.Getenv("LINEAR_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
