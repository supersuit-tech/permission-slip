package providers

import (
	"os"

	dropboxconnector "github.com/supersuit-tech/permission-slip/connectors/dropbox"
	"github.com/supersuit-tech/permission-slip/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "dropbox",
			AuthorizeURL: "https://www.dropbox.com/oauth2/authorize",
			TokenURL:     "https://api.dropboxapi.com/oauth2/token",
			Scopes:       dropboxconnector.OAuthScopes,
			// Dropbox defaults to short-lived tokens; request offline access
			// to get refresh tokens.
			AuthorizeParams: map[string]string{
				"token_access_type": "offline",
			},
			PKCE:         true,
			ClientID:     os.Getenv("DROPBOX_CLIENT_ID"),
			ClientSecret: os.Getenv("DROPBOX_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
