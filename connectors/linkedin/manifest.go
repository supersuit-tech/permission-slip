package linkedin

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
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
							"description": "Post text content (max 3,000 characters)",
							"x-ui": {"label": "Post text", "widget": "textarea"}
						},
						"visibility": {
							"type": "string",
							"enum": ["PUBLIC", "CONNECTIONS"],
							"default": "PUBLIC",
							"description": "Post visibility (PUBLIC or CONNECTIONS, defaults to PUBLIC)",
							"x-ui": {"label": "Visibility", "widget": "select"}
						},
						"article_url": {
							"type": "string",
							"description": "URL for a link share attachment",
							"x-ui": {"label": "Article URL", "placeholder": "https://example.com/article"}
						},
						"article_title": {
							"type": "string",
							"description": "Title for the link share attachment",
							"x-ui": {"label": "Article title", "placeholder": "My article title"}
						},
						"article_description": {
							"type": "string",
							"description": "Description for the link share attachment",
							"x-ui": {"label": "Article description", "placeholder": "Brief description of the article"}
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
							"description": "The post URN (e.g. 'urn:li:share:123456')",
							"x-ui": {"label": "Post URN", "placeholder": "urn:li:share:1234567890", "help_text": "LinkedIn URN — find in the post URL or API response"}
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
							"description": "The post URN to comment on (e.g. 'urn:li:share:123456')",
							"x-ui": {"label": "Post URN", "placeholder": "urn:li:share:1234567890", "help_text": "LinkedIn URN — find in the post URL or API response"}
						},
						"text": {
							"type": "string",
							"maxLength": 1250,
							"description": "Comment text (max 1,250 characters)",
							"x-ui": {"label": "Comment", "widget": "textarea"}
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
							"description": "The post URN to get analytics for (e.g. 'urn:li:share:123456')",
							"x-ui": {"label": "Post URN", "placeholder": "urn:li:share:1234567890", "help_text": "LinkedIn URN — find in the post URL or API response"}
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
							"description": "The LinkedIn organization ID (numeric, e.g. '12345')",
							"x-ui": {"label": "Organization ID", "help_text": "Numeric LinkedIn organization ID — find in your Company Page admin URL"}
						},
						"text": {
							"type": "string",
							"maxLength": 3000,
							"description": "Post text content (max 3,000 characters)",
							"x-ui": {"label": "Post text", "widget": "textarea"}
						},
						"visibility": {
							"type": "string",
							"enum": ["PUBLIC"],
							"default": "PUBLIC",
							"description": "Post visibility (company posts are always PUBLIC)",
							"x-ui": {"label": "Visibility", "widget": "select"}
						},
						"article_url": {
							"type": "string",
							"description": "URL for a link share attachment",
							"x-ui": {"label": "Article URL", "placeholder": "https://example.com/article"}
						},
						"article_title": {
							"type": "string",
							"description": "Title for the link share attachment",
							"x-ui": {"label": "Article title", "placeholder": "My article title"}
						}
					}
				}`)),
			},
			{
				ActionType:  "linkedin.send_message",
				Name:        "Send Message",
				Description: "Send a direct message to a LinkedIn connection. Requires Messaging on behalf of members partner privilege and w_messages scope.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["recipient_urn", "body"],
					"properties": {
						"recipient_urn": {
							"type": "string",
							"description": "LinkedIn person URN of the recipient (e.g. 'urn:li:person:123456')",
							"x-ui": {"label": "Recipient", "placeholder": "urn:li:person:abc123", "help_text": "LinkedIn member URN"}
						},
						"subject": {
							"type": "string",
							"maxLength": 200,
							"description": "Message subject line (optional, max 200 characters)",
							"x-ui": {"label": "Subject", "placeholder": "Message subject"}
						},
						"body": {
							"type": "string",
							"maxLength": 8000,
							"description": "Message body text (max 8,000 characters)",
							"x-ui": {"label": "Message", "widget": "textarea"}
						}
					}
				}`)),
			},
			{
				ActionType:  "linkedin.search_people",
				Name:        "Search People",
				Description: "Search LinkedIn members by name, company, or title. Requires Marketing Developer Platform or Sales Navigator API access.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"keywords": {
							"type": "string",
							"description": "Name or general keywords to search for",
							"x-ui": {"label": "Keywords", "placeholder": "software engineer"}
						},
						"company": {
							"type": "string",
							"description": "Filter by current company name or ID",
							"x-ui": {"label": "Company", "placeholder": "Acme Corp"}
						},
						"title": {
							"type": "string",
							"description": "Filter by job title",
							"x-ui": {"label": "Job title", "placeholder": "Engineering Manager"}
						},
						"count": {
							"type": "integer",
							"minimum": 0,
							"maximum": 50,
							"default": 10,
							"description": "Number of results to return (max 50, default 10, 0 = use default)",
							"x-ui": {"label": "Max results"}
						},
						"start": {
							"type": "integer",
							"minimum": 0,
							"default": 0,
							"description": "Pagination offset",
							"x-ui": {"hidden": true}
						}
					}
				}`)),
			},
			{
				ActionType:  "linkedin.search_companies",
				Name:        "Search Companies",
				Description: "Search LinkedIn company pages by keyword. Requires Marketing Developer Platform access.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["keywords"],
					"properties": {
						"keywords": {
							"type": "string",
							"description": "Company name or keywords to search for",
						"x-ui": {"label": "Keywords", "placeholder": "Acme Corp"}
						},
						"count": {
							"type": "integer",
							"minimum": 0,
							"maximum": 50,
							"default": 10,
							"description": "Number of results to return (max 50, default 10, 0 = use default)",
							"x-ui": {"label": "Max results"}
						},
						"start": {
							"type": "integer",
							"minimum": 0,
							"default": 0,
							"description": "Pagination offset",
							"x-ui": {"hidden": true}
						}
					}
				}`)),
			},
			{
				ActionType:  "linkedin.get_company",
				Name:        "Get Company",
				Description: "Get a LinkedIn company profile for enrichment. Requires r_organization_social scope; some fields may need Marketing Developer Platform access.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["organization_id"],
					"properties": {
						"organization_id": {
							"type": "string",
							"description": "The LinkedIn organization ID (numeric, e.g. '12345')",
							"x-ui": {"label": "Organization ID", "help_text": "Numeric LinkedIn organization ID — find in your Company Page admin URL"}
						}
					}
				}`)),
			},
			{
				ActionType:  "linkedin.list_connections",
				Name:        "List Connections",
				Description: "List the authenticated user's LinkedIn connections. Requires r_network scope (restricted; needs LinkedIn partner approval).",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"count": {
							"type": "integer",
							"minimum": 0,
							"maximum": 500,
							"default": 20,
							"description": "Number of connections to return (max 500, default 20, 0 = use default)",
							"x-ui": {"label": "Max results"}
						},
						"start": {
							"type": "integer",
							"minimum": 0,
							"default": 0,
							"description": "Pagination offset",
							"x-ui": {"hidden": true}
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
					// Note: w_messages and r_network require LinkedIn Partner Program
					// approval and are NOT included here. Apps without that approval
					// would fail the OAuth flow entirely if these scopes were present.
					// send_message and list_connections document their scope requirements
					// in their action descriptions; callers using those actions must
					// obtain partner approval and request those scopes separately.
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
			{
				ID:          "tpl_linkedin_send_message",
				ActionType:  "linkedin.send_message",
				Name:        "Send direct messages",
				Description: "Agent can send direct messages to any of your LinkedIn connections.",
				Parameters:  json.RawMessage(`{"recipient_urn":"*","subject":"*","body":"*"}`),
			},
			{
				ID:          "tpl_linkedin_search_people",
				ActionType:  "linkedin.search_people",
				Name:        "Search people on LinkedIn",
				Description: "Agent can search LinkedIn members by keywords, company, or title.",
				Parameters:  json.RawMessage(`{"keywords":"*","company":"*","title":"*","count":"*","start":"*"}`),
			},
			{
				ID:          "tpl_linkedin_search_companies",
				ActionType:  "linkedin.search_companies",
				Name:        "Search companies on LinkedIn",
				Description: "Agent can search LinkedIn company pages by keyword.",
				Parameters:  json.RawMessage(`{"keywords":"*","count":"*","start":"*"}`),
			},
			{
				ID:          "tpl_linkedin_get_company",
				ActionType:  "linkedin.get_company",
				Name:        "Look up company profiles",
				Description: "Agent can retrieve LinkedIn company profile details for enrichment.",
				Parameters:  json.RawMessage(`{"organization_id":"*"}`),
			},
			{
				ID:          "tpl_linkedin_list_connections",
				ActionType:  "linkedin.list_connections",
				Name:        "List your connections",
				Description: "Agent can list your LinkedIn connections with pagination.",
				Parameters:  json.RawMessage(`{"count":"*","start":"*"}`),
			},
		},
	}
}
