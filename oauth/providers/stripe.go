package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "stripe",
			AuthorizeURL: "https://connect.stripe.com/oauth/authorize",
			TokenURL:     "https://connect.stripe.com/oauth/token",
			Scopes: []string{
				"read_write",
			},
			ClientID:     os.Getenv("STRIPE_CLIENT_ID"),
			ClientSecret: os.Getenv("STRIPE_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
