package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "airtable",
			AuthorizeURL: "https://airtable.com/oauth2/v1/authorize",
			TokenURL:     "https://airtable.com/oauth2/v1/token",
			Scopes: []string{
				"data.records:read",
				"data.records:write",
				"data.recordComments:read",
				"data.recordComments:write",
				"schema.bases:read",
				"schema.bases:write",
			},
			ClientID:     os.Getenv("AIRTABLE_CLIENT_ID"),
			ClientSecret: os.Getenv("AIRTABLE_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
