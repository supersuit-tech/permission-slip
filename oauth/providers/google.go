package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			// prompt=consent forces Google to show the consent screen on every
			// authorization, which ensures a refresh token is always returned.
			// Without this, Google only returns a refresh token on the first
			// authorization — subsequent re-authorizations silently omit it,
			// leaving the connection without a way to refresh and forcing the
			// user into a repeated re-auth loop.
			ID:           "google",
			AuthorizeURL: "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			Scopes: []string{
				"openid",
				"https://www.googleapis.com/auth/userinfo.email",
			},
			AuthorizeParams: map[string]string{
				"prompt": "consent",
			},
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
