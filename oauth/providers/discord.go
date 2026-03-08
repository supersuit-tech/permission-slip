package providers

import (
	"os"

	discordconnector "github.com/supersuit-tech/permission-slip-web/connectors/discord"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			ID:           "discord",
			AuthorizeURL: "https://discord.com/oauth2/authorize",
			TokenURL:     "https://discord.com/api/oauth2/token",
			Scopes:       discordconnector.OAuthScopes,
			ClientID:     os.Getenv("DISCORD_CLIENT_ID"),
			ClientSecret: os.Getenv("DISCORD_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
