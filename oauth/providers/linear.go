package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(oauth.Provider{
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
	})
	connectors.RegisterBuiltInOAuthProvider("linear")
}
