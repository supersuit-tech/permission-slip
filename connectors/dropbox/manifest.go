package dropbox

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

//go:embed logo.svg
var logoSVG string

func (c *DropboxConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "dropbox",
		Name:        "Dropbox",
		Description: "Dropbox integration for file management — upload, download, organize, search, and share files",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "dropbox.upload_file",
				Name:        "Upload File",
				Description: "Upload a file to Dropbox",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["path", "content"],
					"properties": {
						"path": {
							"type": "string",
							"description": "Destination path in Dropbox (e.g. /Documents/report.pdf)"
						},
						"content": {
							"type": "string",
							"description": "Base64-encoded file content"
						},
						"mode": {
							"type": "string",
							"enum": ["add", "overwrite"],
							"default": "add",
							"description": "Write mode: add (fail on conflict unless autorename) or overwrite"
						},
						"autorename": {
							"type": "boolean",
							"default": true,
							"description": "Automatically rename on conflict (e.g. file (1).txt)"
						}
					}
				}`)),
			},
			{
				ActionType:  "dropbox.download_file",
				Name:        "Download File",
				Description: "Download file content from Dropbox",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["path"],
					"properties": {
						"path": {
							"type": "string",
							"description": "Path of the file to download (e.g. /Documents/report.pdf)"
						}
					}
				}`)),
			},
			{
				ActionType:  "dropbox.create_folder",
				Name:        "Create Folder",
				Description: "Create a new folder in Dropbox",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["path"],
					"properties": {
						"path": {
							"type": "string",
							"description": "Path of the folder to create (e.g. /Projects/Q1)"
						},
						"autorename": {
							"type": "boolean",
							"default": false,
							"description": "Automatically rename if a folder with this name already exists"
						}
					}
				}`)),
			},
			{
				ActionType:  "dropbox.share_file",
				Name:        "Share File",
				Description: "Create a shared link for a file or folder in Dropbox",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["path"],
					"properties": {
						"path": {
							"type": "string",
							"description": "Path of the file or folder to share (e.g. /Documents/report.pdf)"
						},
						"requested_visibility": {
							"type": "string",
							"enum": ["public", "team_only", "password"],
							"description": "Visibility of the shared link"
						},
						"link_password": {
							"type": "string",
							"description": "Password for the shared link (required when visibility is password)"
						},
						"expires": {
							"type": "string",
							"description": "Expiration date for the shared link (ISO 8601 format)"
						}
					}
				}`)),
			},
			{
				ActionType:  "dropbox.search",
				Name:        "Search",
				Description: "Search for files and folders in Dropbox by name or content",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Search query string"
						},
						"path": {
							"type": "string",
							"description": "Scope search to a specific folder (default: root)"
						},
						"max_results": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of results to return (1-1000)"
						},
						"file_extensions": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Filter by file extensions (e.g. [\"pdf\", \"docx\"])"
						}
					}
				}`)),
			},
			{
				ActionType:  "dropbox.move",
				Name:        "Move/Rename",
				Description: "Move or rename a file or folder in Dropbox",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["from_path", "to_path"],
					"properties": {
						"from_path": {
							"type": "string",
							"description": "Current path of the file or folder (e.g. /old/path.txt)"
						},
						"to_path": {
							"type": "string",
							"description": "New path for the file or folder (e.g. /new/path.txt)"
						},
						"autorename": {
							"type": "boolean",
							"default": false,
							"description": "Automatically rename if a file with this name already exists at the destination"
						},
						"allow_ownership_transfer": {
							"type": "boolean",
							"default": false,
							"description": "Allow moves that result in ownership transfer for the content being moved"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "dropbox",
				AuthType:      "oauth2",
				OAuthProvider: "dropbox",
				OAuthScopes:   OAuthScopes,
			},
		},
		OAuthProviders: []connectors.ManifestOAuthProvider{
			{
				ID:           "dropbox",
				AuthorizeURL: "https://www.dropbox.com/oauth2/authorize",
				TokenURL:     "https://api.dropboxapi.com/oauth2/token",
				Scopes:       OAuthScopes,
				AuthorizeParams: map[string]string{
					"token_access_type": "offline",
				},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_dropbox_upload_file",
				ActionType:  "dropbox.upload_file",
				Name:        "Upload files",
				Description: "Agent can upload files to any Dropbox path.",
				Parameters:  json.RawMessage(`{"path":"*","content":"*","mode":"*","autorename":"*"}`),
			},
			{
				ID:          "tpl_dropbox_download_file",
				ActionType:  "dropbox.download_file",
				Name:        "Download files",
				Description: "Agent can download files from any Dropbox path.",
				Parameters:  json.RawMessage(`{"path":"*"}`),
			},
			{
				ID:          "tpl_dropbox_create_folder",
				ActionType:  "dropbox.create_folder",
				Name:        "Create folders",
				Description: "Agent can create folders in Dropbox.",
				Parameters:  json.RawMessage(`{"path":"*","autorename":"*"}`),
			},
			{
				ID:          "tpl_dropbox_share_file",
				ActionType:  "dropbox.share_file",
				Name:        "Share files",
				Description: "Agent can create shared links for Dropbox files.",
				Parameters:  json.RawMessage(`{"path":"*","requested_visibility":"*","link_password":"*","expires":"*"}`),
			},
			{
				ID:          "tpl_dropbox_search",
				ActionType:  "dropbox.search",
				Name:        "Search files",
				Description: "Agent can search for files in Dropbox.",
				Parameters:  json.RawMessage(`{"query":"*","path":"*","max_results":"*","file_extensions":"*"}`),
			},
			{
				ID:          "tpl_dropbox_move",
				ActionType:  "dropbox.move",
				Name:        "Move/rename files",
				Description: "Agent can move or rename files and folders in Dropbox.",
				Parameters:  json.RawMessage(`{"from_path":"*","to_path":"*","autorename":"*","allow_ownership_transfer":"*"}`),
			},
		},
	}
}
