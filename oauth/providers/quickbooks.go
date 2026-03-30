package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip/oauth"
)

func init() {
	if oauth.IsBuiltInOAuthProviderDisabled("quickbooks") {
		return
	}
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "quickbooks",
			AuthorizeURL: "https://appcenter.intuit.com/connect/oauth2",
			TokenURL:     "https://oauth.platform.intuit.com/oauth2/v1/tokens/bearer",
			Scopes: []string{
				"com.intuit.quickbooks.accounting",
			},
			ClientID:     os.Getenv("QUICKBOOKS_CLIENT_ID"),
			ClientSecret: os.Getenv("QUICKBOOKS_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
