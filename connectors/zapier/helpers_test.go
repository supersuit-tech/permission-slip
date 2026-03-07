package zapier

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with a valid webhook URL for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"webhook_url": "https://hooks.zapier.com/hooks/catch/123456/abcdef/",
	})
}

// credsWithURL returns a Credentials value with a custom webhook URL.
func credsWithURL(url string) connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"webhook_url": url,
	})
}
