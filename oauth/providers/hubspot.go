package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "hubspot",
			AuthorizeURL: "https://app.hubspot.com/oauth/authorize",
			TokenURL:     "https://api.hubapi.com/oauth/v1/token",
			Scopes: []string{
				"crm.objects.contacts.read",
				"crm.objects.contacts.write",
				"crm.objects.deals.read",
				"crm.objects.deals.write",
				"crm.objects.companies.read",
				"tickets",
				"automation",
				"content",
				"analytics.read",
			},
			ClientID:     os.Getenv("HUBSPOT_CLIENT_ID"),
			ClientSecret: os.Getenv("HUBSPOT_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
