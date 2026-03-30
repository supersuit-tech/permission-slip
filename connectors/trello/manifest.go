// This file is split from trello.go to keep the connector file focused on
// struct, auth, and HTTP lifecycle. Action schemas live in manifest_actions.go
// and templates live in manifest_templates.go.
package trello

import (
	_ "embed"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
//go:embed logo.svg
var logoSVG string

func (c *TrelloConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "trello",
		Name:        "Trello",
		Description: "Trello integration for project management and kanban boards",
		LogoSVG:     logoSVG,
		OAuthProviders: []connectors.ManifestOAuthProvider{
			{
				ID:           "trello",
				AuthorizeURL: "https://auth.atlassian.com/authorize",
				TokenURL:     "https://auth.atlassian.com/oauth/token",
				Scopes:       []string{"read:me:trello", "read:board:trello", "write:board:trello"},
				AuthorizeParams: map[string]string{
					"audience": "api.atlassian.com",
					"prompt":   "consent",
				},
			},
		},
		Actions: trelloActions(),
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "trello_oauth",
				AuthType:      "oauth2",
				OAuthProvider: "trello",
				OAuthScopes:   []string{"read:me:trello", "read:board:trello", "write:board:trello"},
			},
			{
				Service:         "trello",
				AuthType:        "api_key",
				InstructionsURL: "https://developer.atlassian.com/cloud/trello/guides/rest-api/api-introduction/#authentication-and-authorization",
			},
		},
		Templates: trelloTemplates(),
	}
}
