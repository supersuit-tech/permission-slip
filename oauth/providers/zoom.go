package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "zoom",
			AuthorizeURL: "https://zoom.us/oauth/authorize",
			TokenURL:     "https://zoom.us/oauth/token",
			Scopes: []string{
				"meeting:read",
				"meeting:write",
				"recording:read",
				"user:read",
			},
			ClientID:     os.Getenv("ZOOM_CLIENT_ID"),
			ClientSecret: os.Getenv("ZOOM_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
