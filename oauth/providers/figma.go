package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(oauth.Provider{
		ID:           "figma",
		AuthorizeURL: "https://www.figma.com/oauth",
		TokenURL:     "https://api.figma.com/v1/oauth/token",
		Scopes: []string{
			"files:read",
			"file_comments:write",
		},
		ClientID:     os.Getenv("FIGMA_CLIENT_ID"),
		ClientSecret: os.Getenv("FIGMA_CLIENT_SECRET"),
		Source:       oauth.SourceBuiltIn,
	})
	connectors.RegisterBuiltInOAuthProvider("figma")
}
