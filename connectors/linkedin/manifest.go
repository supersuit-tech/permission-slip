package linkedin

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
//go:embed logo.svg
var logoSVG string

func (c *LinkedInConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "linkedin",
		Name:        "LinkedIn",
		Description: "LinkedIn integration for professional social media management",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "linkedin.create_post",
				Name:        "Create Post",
				Description: "Create a post on the authenticated user's LinkedIn feed",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["text"],
					"properties": {
						"text": {
							"type": "string",
							"maxLength": 3000,
							"description": "Post text content (max 3,000 characters)"
						},
						"visibility": {
							"type": "string",
							"enum": ["PUBLIC", "CONNECTIONS"],
							"default": "PUBLIC",
							"description": "Post visibility (PUBLIC or CONNECTIONS, defaults to PUBLIC)"
						},
						"article_url": {
							"type": "string",
							"description": "URL for a link share attachment"
						},
						"article_title": {
							"type": "string",
							"description": "Title for the link share attachment"
						},
						"article_description": {
							"type": "string",
							"description": "Description for the link share attachment"
						}
					}
				}`)),
			},
			{
				ActionType:  "linkedin.delete_post",
				Name:        "Delete Post",
				Description: "Delete a post from LinkedIn (irreversible)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["post_urn"],
					"properties": {
						"post_urn": {
							"type": "string",
							"description": "The post URN (e.g. 'urn:li:share:123456')"
						}
					}
				}`)),
			},
			{
				ActionType:  "linkedin.add_comment",
				Name:        "Add Comment",
				Description: "Add a comment on a LinkedIn post",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["post_urn", "text"],
					"properties": {
						"post_urn": {
							"type": "string",
							"description": "The post URN to comment on (e.g. 'urn:li:share:123456')"
						},
						"text": {
							"type": "string",
							"maxLength": 1250,
							"description": "Comment text (max 1,250 characters)"
						}
					}
				}`)),
			},
			{
				ActionType:  "linkedin.get_profile",
				Name:        "Get My Profile",
				Description: "Get the authenticated user's LinkedIn profile",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:  "linkedin.get_post_analytics",
				Name:        "Get Post Analytics",
				Description: "Get engagement metrics for a specific LinkedIn post",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["post_urn"],
					"properties": {
						"post_urn": {
							"type": "string",
							"description": "The post URN to get analytics for (e.g. 'urn:li:share:123456')"
						}
					}
				}`)),
			},
			{
				ActionType:  "linkedin.create_company_post",
				Name:        "Create Company Post",
				Description: "Post on behalf of a company page the user administers",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["organization_id", "text"],
					"properties": {
						"organization_id": {
							"type": "string",
							"description": "The LinkedIn organization ID (numeric, e.g. '12345')"
						},
						"text": {
							"type": "string",
							"maxLength": 3000,
							"description": "Post text content (max 3,000 characters)"
						},
						"visibility": {
							"type": "string",
							"enum": ["PUBLIC"],
							"default": "PUBLIC",
							"description": "Post visibility (company posts are always PUBLIC)"
						},
						"article_url": {
							"type": "string",
							"description": "URL for a link share attachment"
						},
						"article_title": {
							"type": "string",
							"description": "Title for the link share attachment"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "linkedin",
				AuthType:      "oauth2",
				OAuthProvider: "linkedin",
				OAuthScopes: []string{
					"openid",
					"profile",
					"w_member_social",
					"r_organization_social",
					"w_organization_social",
				},
				InstructionsURL: "https://learn.microsoft.com/en-us/linkedin/shared/authentication/authorization-code-flow",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_linkedin_create_post",
				ActionType:  "linkedin.create_post",
				Name:        "Post updates on LinkedIn",
				Description: "Agent can create posts with any text and visibility on your LinkedIn feed.",
				Parameters:  json.RawMessage(`{"text":"*","visibility":"*"}`),
			},
			{
				ID:          "tpl_linkedin_create_post_public",
				ActionType:  "linkedin.create_post",
				Name:        "Post public updates",
				Description: "Agent can create public posts on your LinkedIn feed.",
				Parameters:  json.RawMessage(`{"text":"*","visibility":"PUBLIC"}`),
			},
			{
				ID:          "tpl_linkedin_create_post_with_link",
				ActionType:  "linkedin.create_post",
				Name:        "Share links on LinkedIn",
				Description: "Agent can create posts with link attachments on your LinkedIn feed.",
				Parameters:  json.RawMessage(`{"text":"*","visibility":"*","article_url":"*","article_title":"*","article_description":"*"}`),
			},
			{
				ID:          "tpl_linkedin_delete_post",
				ActionType:  "linkedin.delete_post",
				Name:        "Delete posts",
				Description: "Agent can delete any of your LinkedIn posts.",
				Parameters:  json.RawMessage(`{"post_urn":"*"}`),
			},
			{
				ID:          "tpl_linkedin_add_comment",
				ActionType:  "linkedin.add_comment",
				Name:        "Comment on posts",
				Description: "Agent can add comments on any LinkedIn post.",
				Parameters:  json.RawMessage(`{"post_urn":"*","text":"*"}`),
			},
			{
				ID:          "tpl_linkedin_get_profile",
				ActionType:  "linkedin.get_profile",
				Name:        "View profile",
				Description: "Agent can view your LinkedIn profile information.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_linkedin_get_post_analytics",
				ActionType:  "linkedin.get_post_analytics",
				Name:        "View post analytics",
				Description: "Agent can view engagement metrics for any LinkedIn post.",
				Parameters:  json.RawMessage(`{"post_urn":"*"}`),
			},
			{
				ID:          "tpl_linkedin_create_company_post",
				ActionType:  "linkedin.create_company_post",
				Name:        "Post to company page",
				Description: "Agent can post on behalf of a specific company page.",
				Parameters:  json.RawMessage(`{"organization_id":"*","text":"*","visibility":"PUBLIC"}`),
			},
			{
				ID:          "tpl_linkedin_create_company_post_specific",
				ActionType:  "linkedin.create_company_post",
				Name:        "Post to specific company page",
				Description: "Locks the organization; agent chooses the post content.",
				Parameters:  json.RawMessage(`{"organization_id":"ORG_ID_HERE","text":"*","visibility":"PUBLIC"}`),
			},
		},
	}
}
