package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(oauth.Provider{
		ID:           "microsoft",
		AuthorizeURL: "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
		TokenURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/token",
		Scopes: []string{
			"openid",
			"offline_access",
			"User.Read",
		},
		ClientID:     os.Getenv("MICROSOFT_CLIENT_ID"),
		ClientSecret: os.Getenv("MICROSOFT_CLIENT_SECRET"),
		Source:       oauth.SourceBuiltIn,
	})
	connectors.RegisterBuiltInOAuthProvider("microsoft")
}
