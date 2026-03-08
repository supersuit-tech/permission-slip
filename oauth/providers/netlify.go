package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(oauth.Provider{
		ID:           "netlify",
		AuthorizeURL: "https://app.netlify.com/authorize",
		TokenURL:     "https://api.netlify.com/oauth/token",
		ClientID:     os.Getenv("NETLIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("NETLIFY_CLIENT_SECRET"),
		Source:       oauth.SourceBuiltIn,
	})
	connectors.RegisterBuiltInOAuthProvider("netlify")
}
