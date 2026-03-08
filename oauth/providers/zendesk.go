package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			// Zendesk uses per-subdomain OAuth URLs. The {subdomain} placeholder
			// is replaced at authorize/callback time with the user's Zendesk
			// subdomain (e.g. "mycompany"). See api/oauth.go for resolution.
			ID:           "zendesk",
			AuthorizeURL: "https://{subdomain}.zendesk.com/oauth/authorizations/new",
			TokenURL:     "https://{subdomain}.zendesk.com/oauth/tokens",
			Scopes: []string{
				"read",
				"write",
			},
			ClientID:     os.Getenv("ZENDESK_CLIENT_ID"),
			ClientSecret: os.Getenv("ZENDESK_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
