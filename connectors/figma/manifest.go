package figma

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//go:embed logo.svg
var logoSVG string

func (c *FigmaConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "figma",
		Name:        "Figma",
		Description: "Figma integration for design file access and collaboration",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "figma.get_file",
				Name:        "Get File",
				Description: "Get a full design file tree with metadata",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key or full Figma URL. You can paste a URL like https://www.figma.com/design/abc123DEF/... and the key will be extracted automatically."
						},
						"depth": {
							"type": "integer",
							"description": "Positive integer specifying how deep to traverse the document tree. Omit for full depth."
						},
						"node_ids": {
							"type": "string",
							"description": "Comma-separated list of node IDs to retrieve (e.g. \"1:2,3:4\"). Returns only those subtrees."
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.get_components",
				Name:        "Get Components",
				Description: "Get design system components from a file",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key or full Figma URL (key is extracted automatically from URLs)"
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.export_images",
				Name:        "Export Images",
				Description: "Export PNG, SVG, PDF, or JPG from specific nodes in a file",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key", "node_ids", "format"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key or full Figma URL (key is extracted automatically from URLs)"
						},
						"node_ids": {
							"type": "string",
							"description": "Comma-separated list of node IDs to export (e.g. \"1:2,3:4\")"
						},
						"format": {
							"type": "string",
							"enum": ["png", "svg", "pdf", "jpg"],
							"description": "Image export format"
						},
						"scale": {
							"type": "number",
							"minimum": 0.01,
							"maximum": 4,
							"default": 1,
							"description": "Image scale factor (0.01–4, default 1). Only applies to png/jpg."
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.list_comments",
				Name:        "List Comments",
				Description: "List comments on a Figma file",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key or full Figma URL (key is extracted automatically from URLs)"
						},
						"as_md": {
							"type": "boolean",
							"default": false,
							"description": "If true, return comment messages as markdown"
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.post_comment",
				Name:        "Post Comment",
				Description: "Post a comment on a Figma file",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key", "message"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key or full Figma URL (key is extracted automatically from URLs)"
						},
						"message": {
							"type": "string",
							"description": "Comment message text"
						},
						"comment_id": {
							"type": "string",
							"description": "ID of the comment to reply to (for threaded replies)"
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.get_versions",
				Name:        "Get Versions",
				Description: "Get the version history for a Figma file",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key or full Figma URL (key is extracted automatically from URLs)"
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.get_styles",
				Name:        "Get Styles",
				Description: "Get design styles/tokens (colors, text, effects) from a Figma file — the foundation of design-to-code workflows",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key or full Figma URL (key is extracted automatically from URLs)"
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.list_projects",
				Name:        "List Projects",
				Description: "List projects in a Figma team",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["team_id"],
					"properties": {
						"team_id": {
							"type": "string",
							"description": "The Figma team ID"
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.list_files",
				Name:        "List Files",
				Description: "List files in a Figma project",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_id"],
					"properties": {
						"project_id": {
							"type": "string",
							"description": "The project ID to list files from"
						}
					}
				}`)),
			},
			{
				ActionType:  "figma.get_variables",
				Name:        "Get Variables",
				Description: "Get design system variables (tokens) from a Figma file — supports multi-mode values for light/dark, brand themes, etc.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["file_key"],
					"properties": {
						"file_key": {
							"type": "string",
							"description": "The file key or full Figma URL (key is extracted automatically from URLs)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "figma",
				AuthType:        "oauth2",
				OAuthProvider:   "figma",
				OAuthScopes:     []string{"files:read", "file_comments:write"},
				InstructionsURL: "https://help.figma.com/hc/en-us/articles/8085703771159-Manage-personal-access-tokens",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_figma_get_file",
				ActionType:  "figma.get_file",
				Name:        "Read design file",
				Description: "Agent can read any Figma file's design tree and metadata.",
				Parameters:  json.RawMessage(`{"file_key":"*","depth":"*","node_ids":"*"}`),
			},
			{
				ID:          "tpl_figma_get_components",
				ActionType:  "figma.get_components",
				Name:        "Get design components",
				Description: "Agent can list components from any Figma file.",
				Parameters:  json.RawMessage(`{"file_key":"*"}`),
			},
			{
				ID:          "tpl_figma_export_images",
				ActionType:  "figma.export_images",
				Name:        "Export images from designs",
				Description: "Agent can export images from any Figma file nodes.",
				Parameters:  json.RawMessage(`{"file_key":"*","node_ids":"*","format":"*","scale":"*"}`),
			},
			{
				ID:          "tpl_figma_list_comments",
				ActionType:  "figma.list_comments",
				Name:        "Read file comments",
				Description: "Agent can list comments on any Figma file.",
				Parameters:  json.RawMessage(`{"file_key":"*","as_md":"*"}`),
			},
			{
				ID:          "tpl_figma_post_comment",
				ActionType:  "figma.post_comment",
				Name:        "Post comments on designs",
				Description: "Agent can post comments on any Figma file.",
				Parameters:  json.RawMessage(`{"file_key":"*","message":"*","comment_id":"*"}`),
			},
			{
				ID:          "tpl_figma_get_versions",
				ActionType:  "figma.get_versions",
				Name:        "View version history",
				Description: "Agent can view version history of any Figma file.",
				Parameters:  json.RawMessage(`{"file_key":"*"}`),
			},
			{
				ID:          "tpl_figma_get_styles",
				ActionType:  "figma.get_styles",
				Name:        "Get design styles",
				Description: "Agent can get design styles/tokens from any Figma file.",
				Parameters:  json.RawMessage(`{"file_key":"*"}`),
			},
			{
				ID:          "tpl_figma_list_projects",
				ActionType:  "figma.list_projects",
				Name:        "List team projects",
				Description: "Agent can list projects in a Figma team.",
				Parameters:  json.RawMessage(`{"team_id":"*"}`),
			},
			{
				ID:          "tpl_figma_list_files",
				ActionType:  "figma.list_files",
				Name:        "List project files",
				Description: "Agent can list files in any Figma project.",
				Parameters:  json.RawMessage(`{"project_id":"*"}`),
			},
			{
				ID:          "tpl_figma_get_variables",
				ActionType:  "figma.get_variables",
				Name:        "Get design variables",
				Description: "Agent can get design system variables from any Figma file.",
				Parameters:  json.RawMessage(`{"file_key":"*"}`),
			},
		},
	}
}
