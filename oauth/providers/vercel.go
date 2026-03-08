package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(oauth.Provider{
		ID:           "vercel",
		AuthorizeURL: "https://vercel.com/oauth/authorize",
		TokenURL:     "https://api.vercel.com/v2/oauth/access_token",
		ClientID:     os.Getenv("VERCEL_CLIENT_ID"),
		ClientSecret: os.Getenv("VERCEL_CLIENT_SECRET"),
		Source:       oauth.SourceBuiltIn,
	})
	connectors.RegisterBuiltInOAuthProvider("vercel")
}
