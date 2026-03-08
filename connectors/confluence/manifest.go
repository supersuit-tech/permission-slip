package confluence

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//go:embed logo.svg
var logoSVG string

func (c *ConfluenceConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "confluence",
		Name:        "Confluence",
		Description: "Confluence Cloud integration for documentation and knowledge management",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "confluence.create_page",
				Name:        "Create Page",
				Description: "Create a new page in a Confluence space",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["space_id", "title", "body"],
					"properties": {
						"space_id": {
							"type": "string",
							"description": "ID of the space to create the page in"
						},
						"title": {
							"type": "string",
							"description": "Page title"
						},
						"body": {
							"type": "string",
							"description": "Page body content (storage format XHTML or plain text)"
						},
						"parent_id": {
							"type": "string",
							"description": "ID of the parent page (optional, creates at space root if omitted)"
						},
						"status": {
							"type": "string",
							"enum": ["current", "draft"],
							"default": "current",
							"description": "Page status (current or draft)"
						}
					}
				}`)),
			},
			{
				ActionType:  "confluence.update_page",
				Name:        "Update Page",
				Description: "Update the title or content of a Confluence page",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["page_id", "version_number"],
					"properties": {
						"page_id": {
							"type": "string",
							"description": "ID of the page to update"
						},
						"title": {
							"type": "string",
							"description": "Updated page title"
						},
						"body": {
							"type": "string",
							"description": "Updated page body content (storage format, full replacement)"
						},
						"version_number": {
							"type": "integer",
							"description": "Version number for optimistic locking (must be current version + 1)"
						},
						"version_message": {
							"type": "string",
							"description": "Optional message describing the change"
						},
						"status": {
							"type": "string",
							"enum": ["current", "draft"],
							"description": "Page status (current or draft)"
						}
					}
				}`)),
			},
			{
				ActionType:  "confluence.get_page",
				Name:        "Get Page",
				Description: "Get page content and metadata — use before updating to get current version number",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["page_id"],
					"properties": {
						"page_id": {
							"type": "string",
							"description": "ID of the page to retrieve"
						},
						"body_format": {
							"type": "string",
							"enum": ["storage", "atlas_doc_format", "view"],
							"default": "storage",
							"description": "Format for the page body (storage, atlas_doc_format, or view)"
						}
					}
				}`)),
			},
			{
				ActionType:  "confluence.search",
				Name:        "Search",
				Description: "Search across pages using CQL (Confluence Query Language)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["cql"],
					"properties": {
						"cql": {
							"type": "string",
							"description": "CQL query string (e.g. type=page AND space=DEV AND text~\"deployment\")"
						},
						"limit": {
							"type": "integer",
							"default": 25,
							"description": "Maximum number of results to return (max 250)"
						}
					}
				}`)),
			},
			{
				ActionType:  "confluence.add_comment",
				Name:        "Add Comment",
				Description: "Add a footer comment to a Confluence page",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["page_id", "body"],
					"properties": {
						"page_id": {
							"type": "string",
							"description": "ID of the page to comment on"
						},
						"body": {
							"type": "string",
							"description": "Comment body (storage format XHTML or plain text)"
						}
					}
				}`)),
			},
		{
				ActionType:  "confluence.list_spaces",
				Name:        "List Spaces",
				Description: "List available spaces in Confluence",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"limit": {
							"type": "integer",
							"default": 25,
							"minimum": 1,
							"maximum": 250,
							"description": "Maximum number of spaces to return (1-250, default 25)"
						},
						"status": {
							"type": "string",
							"enum": ["current", "archived"],
							"default": "current",
							"description": "Filter by space status (default: current)"
						}
					}
				}`)),
			},
			{
				ActionType:  "confluence.list_pages",
				Name:        "List Pages",
				Description: "List pages in a Confluence space",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["space_id"],
					"properties": {
						"space_id": {
							"type": "string",
							"description": "ID of the space to list pages from"
						},
						"limit": {
							"type": "integer",
							"default": 25,
							"minimum": 1,
							"maximum": 250,
							"description": "Maximum number of pages to return (1-250, default 25)"
						},
						"status": {
							"type": "string",
							"enum": ["current", "archived", "deleted", "trashed"],
							"default": "current",
							"description": "Filter by page status (default: current)"
						}
					}
				}`)),
			},
			{
				ActionType:  "confluence.delete_page",
				Name:        "Delete Page",
				Description: "Delete (move to trash) a Confluence page — can be restored from trash",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["page_id"],
					"properties": {
						"page_id": {
							"type": "string",
							"description": "ID of the page to delete"
						}
					}
				}`)),
			},
			{
				ActionType:  "confluence.get_attachments",
				Name:        "Get Attachments",
				Description: "List attachments on a Confluence page",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["page_id"],
					"properties": {
						"page_id": {
							"type": "string",
							"description": "ID of the page to get attachments for"
						},
						"limit": {
							"type": "integer",
							"default": 25,
							"minimum": 1,
							"maximum": 250,
							"description": "Maximum number of attachments to return (1-250, default 25)"
						},
						"media_type": {
							"type": "string",
							"description": "Filter by MIME type (e.g. 'image/png', 'application/pdf')"
						}
					}
				}`)),
			},
			{
				ActionType:  "confluence.add_attachment",
				Name:        "Add Attachment",
				Description: "Upload a file attachment to a Confluence page. File content must be base64-encoded.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["page_id", "filename", "content_base64"],
					"properties": {
						"page_id": {
							"type": "string",
							"description": "ID of the page to attach the file to"
						},
						"filename": {
							"type": "string",
							"description": "Filename for the attachment (e.g. 'diagram.png')"
						},
						"content_base64": {
							"type": "string",
							"description": "Base64-encoded file content"
						},
						"media_type": {
							"type": "string",
							"description": "MIME type (e.g. 'image/png'). Inferred from filename extension if omitted."
						},
						"comment": {
							"type": "string",
							"description": "Optional comment describing the attachment"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "jira",
				AuthType:        "basic",
				InstructionsURL: "https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_confluence_create_page_space",
				ActionType:  "confluence.create_page",
				Name:        "Create pages in a space",
				Description: "Agent can create pages in a specific Confluence space.",
				Parameters:  json.RawMessage(`{"space_id":"YOUR_SPACE_ID","title":"*","body":"*","parent_id":"*","status":"*"}`),
			},
			{
				ID:          "tpl_confluence_create_page_all",
				ActionType:  "confluence.create_page",
				Name:        "Create pages (all spaces)",
				Description: "Agent can create pages in any Confluence space.",
				Parameters:  json.RawMessage(`{"space_id":"*","title":"*","body":"*","parent_id":"*","status":"*"}`),
			},
			{
				ID:          "tpl_confluence_update_page",
				ActionType:  "confluence.update_page",
				Name:        "Update pages",
				Description: "Agent can update any page's title or content.",
				Parameters:  json.RawMessage(`{"page_id":"*","title":"*","body":"*","version_number":"*","version_message":"*","status":"*"}`),
			},
			{
				ID:          "tpl_confluence_get_page",
				ActionType:  "confluence.get_page",
				Name:        "Read pages",
				Description: "Agent can read any page's content and metadata.",
				Parameters:  json.RawMessage(`{"page_id":"*","body_format":"*"}`),
			},
			{
				ID:          "tpl_confluence_search",
				ActionType:  "confluence.search",
				Name:        "Search pages",
				Description: "Agent can search across Confluence pages using CQL.",
				Parameters:  json.RawMessage(`{"cql":"*","limit":"*"}`),
			},
			{
				ID:          "tpl_confluence_add_comment",
				ActionType:  "confluence.add_comment",
				Name:        "Comment on pages",
				Description: "Agent can add comments to any page.",
				Parameters:  json.RawMessage(`{"page_id":"*","body":"*"}`),
			},
			{
				ID:          "tpl_confluence_list_spaces",
				ActionType:  "confluence.list_spaces",
				Name:        "List spaces",
				Description: "Agent can list all available Confluence spaces.",
				Parameters:  json.RawMessage(`{"limit":"*","status":"*"}`),
			},
			{
				ID:          "tpl_confluence_list_pages",
				ActionType:  "confluence.list_pages",
				Name:        "List pages in a space",
				Description: "Agent can list pages in any Confluence space.",
				Parameters:  json.RawMessage(`{"space_id":"*","limit":"*","status":"*"}`),
			},
			{
				ID:          "tpl_confluence_list_pages_specific",
				ActionType:  "confluence.list_pages",
				Name:        "List pages in specific space",
				Description: "Locks the space; agent can filter by status.",
				Parameters:  json.RawMessage(`{"space_id":"YOUR_SPACE_ID","limit":"*","status":"*"}`),
			},
			{
				ID:          "tpl_confluence_delete_page",
				ActionType:  "confluence.delete_page",
				Name:        "Delete pages",
				Description: "Agent can delete any page (moves to trash).",
				Parameters:  json.RawMessage(`{"page_id":"*"}`),
			},
			{
				ID:          "tpl_confluence_get_attachments",
				ActionType:  "confluence.get_attachments",
				Name:        "List page attachments",
				Description: "Agent can list attachments on any page.",
				Parameters:  json.RawMessage(`{"page_id":"*","limit":"*","media_type":"*"}`),
			},
			{
				ID:          "tpl_confluence_add_attachment",
				ActionType:  "confluence.add_attachment",
				Name:        "Upload attachments to pages",
				Description: "Agent can upload file attachments to any page.",
				Parameters:  json.RawMessage(`{"page_id":"*","filename":"*","content_base64":"*","media_type":"*","comment":"*"}`),
			},
		},
	}
}
