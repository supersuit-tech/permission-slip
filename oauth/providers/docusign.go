package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "docusign",
			AuthorizeURL: "https://account.docusign.com/oauth/auth",
			TokenURL:     "https://account.docusign.com/oauth/token",
			Scopes: []string{
				"signature",
			},
			ClientID:     os.Getenv("DOCUSIGN_CLIENT_ID"),
			ClientSecret: os.Getenv("DOCUSIGN_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
