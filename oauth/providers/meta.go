package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(oauth.Provider{
		ID:           "meta",
		AuthorizeURL: "https://www.facebook.com/v19.0/dialog/oauth",
		TokenURL:     "https://graph.facebook.com/v19.0/oauth/access_token",
		Scopes: []string{
			"pages_manage_posts",
			"pages_read_engagement",
			"pages_read_user_content",
			"instagram_basic",
			"instagram_content_publish",
			"instagram_manage_insights",
		},
		ClientID:     os.Getenv("META_CLIENT_ID"),
		ClientSecret: os.Getenv("META_CLIENT_SECRET"),
		Source:       oauth.SourceBuiltIn,
	})
	connectors.RegisterBuiltInOAuthProvider("meta")
}
