package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	if oauth.IsBuiltInOAuthProviderDisabled("kroger") {
		return
	}
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "kroger",
			AuthorizeURL: "https://api.kroger.com/v1/connect/oauth2/authorize",
			TokenURL:     "https://api.kroger.com/v1/connect/oauth2/token",
			Scopes: []string{
				"product.compact",
				"cart.basic:write",
			},
			ClientID:     os.Getenv("KROGER_CLIENT_ID"),
			ClientSecret: os.Getenv("KROGER_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
