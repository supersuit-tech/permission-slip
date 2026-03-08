package providers

import (
	"os"

	datadogconnector "github.com/supersuit-tech/permission-slip-web/connectors/datadog"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			// Datadog OAuth uses split hostnames by design: authorization
			// redirects go through app.datadoghq.com, while token exchange
			// happens on api.datadoghq.com. Both paths use the /oauth2/v1/
			// prefix and are documented in Datadog's OAuth2 API reference.
			// Scopes are sourced from the connector package to keep the
			// manifest credential and this registration in sync.
			ID:           "datadog",
			AuthorizeURL: "https://app.datadoghq.com/oauth2/v1/authorize",
			TokenURL:     "https://api.datadoghq.com/oauth2/v1/token",
			Scopes:       datadogconnector.OAuthScopes,
			ClientID:     os.Getenv("DATADOG_CLIENT_ID"),
			ClientSecret: os.Getenv("DATADOG_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
