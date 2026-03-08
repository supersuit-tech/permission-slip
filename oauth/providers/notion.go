package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "notion",
			AuthorizeURL: "https://api.notion.com/v1/oauth/authorize",
			TokenURL:     "https://api.notion.com/v1/oauth/token",
			Scopes:       []string{},
			ClientID:     os.Getenv("NOTION_CLIENT_ID"),
			ClientSecret: os.Getenv("NOTION_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
