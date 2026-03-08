package providers

import (
	"os"

	"github.com/supersuit-tech/permission-slip-web/oauth"
)

func init() {
	oauth.RegisterBuiltIn(func() oauth.Provider {
		return oauth.Provider{
			// Atlassian OAuth 2.0 (3LO) covers Jira, Confluence, and other
			// Atlassian Cloud products. The audience param is required by
			// Atlassian's authorization server to issue API-scoped tokens.
			// prompt=consent ensures a refresh token is always returned.
			//
			// Scopes:
			//   read:me          — user profile, required for accessible-resources discovery
			//   read:jira-work   — read issues, projects, search results
			//   write:jira-work  — create/update issues, add comments, transitions
			//   offline_access   — obtain a refresh token for long-lived connections
			ID:           "atlassian",
			AuthorizeURL: "https://auth.atlassian.com/authorize",
			TokenURL:     "https://auth.atlassian.com/oauth/token",
			Scopes: []string{
				"read:me",
				"read:jira-work",
				"write:jira-work",
				"offline_access",
			},
			AuthorizeParams: map[string]string{
				"audience": "api.atlassian.com",
				"prompt":   "consent",
			},
			ClientID:     os.Getenv("ATLASSIAN_CLIENT_ID"),
			ClientSecret: os.Getenv("ATLASSIAN_CLIENT_SECRET"),
			Source:       oauth.SourceBuiltIn,
		}
	})
}
