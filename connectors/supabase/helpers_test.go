package supabase

import "github.com/supersuit-tech/permission-slip-web/connectors"

// validCreds returns a Credentials value with valid Supabase credentials for tests.
func validCreds() connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"supabase_url": "http://localhost:54321",
		"api_key":      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test-key",
	})
}

// validCredsWithURL returns test credentials pointing at a specific base URL.
func validCredsWithURL(baseURL string) connectors.Credentials {
	return connectors.NewCredentials(map[string]string{
		"supabase_url": baseURL,
		"api_key":      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test-key",
	})
}
