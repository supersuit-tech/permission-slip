package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "x",
			AuthorizeURL: "https://x.com/i/oauth2/authorize",
			TokenURL:     "https://api.x.com/2/oauth2/token",
			Scopes: []string{
				"tweet.read",
				"tweet.write",
				"users.read",
				"dm.read",
				"dm.write",
				"offline.access",
				"like.read",
				"like.write",
				"follows.read",
				"follows.write",
			},
			ClientID:     os.Getenv("X_CLIENT_ID"),
			ClientSecret: os.Getenv("X_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
