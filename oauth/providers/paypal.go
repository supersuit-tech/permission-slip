package providers

import (
	"os"

	paypalconnector "github.com/supersuit-tech/permission-slip/connectors/paypal"
	"github.com/supersuit-tech/permission-slip/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			// Log in with PayPal (OpenID). Authorization is on www.paypal.com;
			// token exchange uses api.paypal.com (matches PayPal OpenID discovery).
			ID:           "paypal",
			AuthorizeURL: "https://www.paypal.com/signin/authorize",
			TokenURL:     "https://api.paypal.com/v1/oauth2/token",
			Scopes:       paypalconnector.OAuthScopes,
			ClientID:     os.Getenv("PAYPAL_CLIENT_ID"),
			ClientSecret: os.Getenv("PAYPAL_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
