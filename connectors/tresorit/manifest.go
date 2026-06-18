package tresorit

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

//go:embed logo.svg
var logoSVG string

func (c *TresoritConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "tresorit",
		Name:        "Tresorit",
		Description: "Read and write files in Tresorit via the local S3-compatible API gateway. The gateway Docker container must be running on the same host as Permission Slip.",
		Status:      "early_preview",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "tresorit.list_files",
				Name:        "List Files",
				Description: "List files and folders in a Tresor, optionally filtered by prefix",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tresor"],
					"properties": {
						"tresor": {
							"type": "string",
							"description": "Tresor name (S3 bucket name in the gateway)"
						},
						"prefix": {
							"type": "string",
							"description": "Optional folder prefix to scope the listing"
						},
						"max_keys": {
							"type": "integer",
							"default": 1000,
							"minimum": 1,
							"maximum": 1000,
							"description": "Maximum number of keys to return"
						}
					}
				}`)),
			},
			{
				ActionType:  "tresorit.download_file",
				Name:        "Download File",
				Description: "Download a file from a Tresor (content returned as base64)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tresor", "key"],
					"properties": {
						"tresor": {
							"type": "string",
							"description": "Tresor name"
						},
						"key": {
							"type": "string",
							"description": "File path within the Tresor"
						}
					}
				}`)),
			},
			{
				ActionType:  "tresorit.upload_file",
				Name:        "Upload File",
				Description: "Upload a file to a Tresor (content as base64)",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tresor", "key", "content"],
					"properties": {
						"tresor": {
							"type": "string",
							"description": "Tresor name"
						},
						"key": {
							"type": "string",
							"description": "Destination file path within the Tresor"
						},
						"content": {
							"type": "string",
							"description": "Base64-encoded file content"
						}
					}
				}`)),
			},
			{
				ActionType:  "tresorit.create_folder",
				Name:        "Create Folder",
				Description: "Create a folder inside a Tresor",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tresor", "path"],
					"properties": {
						"tresor": {
							"type": "string",
							"description": "Tresor name"
						},
						"path": {
							"type": "string",
							"description": "Folder path to create (a trailing slash is added if missing)"
						}
					}
				}`)),
			},
			{
				ActionType:  "tresorit.delete_file",
				Name:        "Delete File",
				Description: "Delete a file from a Tresor",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tresor", "key"],
					"properties": {
						"tresor": {
							"type": "string",
							"description": "Tresor name"
						},
						"key": {
							"type": "string",
							"description": "File path to delete"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "tresorit",
				AuthType:        "custom",
				InstructionsURL: "https://github.com/supersuit-tech/permission-slip/blob/main/connectors/tresorit/README.md",
				Fields: []connectors.ManifestCredentialField{
					{
						Key:         credKeyAccessKey,
						Label:       "Access key",
						Placeholder: "client_id from credentials.json",
						Secret:      ptrBool(false),
						Required:    ptrBool(true),
						HelpText:    "The client_id value from the gateway's credentials.json after login.",
					},
					{
						Key:         credKeySecretKey,
						Label:       "Secret key",
						Placeholder: "client_secret from credentials.json",
						Secret:      ptrBool(true),
						Required:    ptrBool(true),
						HelpText:    "The client_secret value from the gateway's credentials.json.",
					},
					{
						Key:         credKeyEndpointURL,
						Label:       "Gateway endpoint",
						Placeholder: "http://127.0.0.1:3000",
						Secret:      ptrBool(false),
						Required:    ptrBool(true),
						HelpText:    "URL of the local Tresorit S3 gateway (default port 3000). Use http:// for loopback unless you terminate TLS locally.",
					},
				},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_tresorit_list",
				ActionType:  "tresorit.list_files",
				Name:        "List files in any Tresor",
				Description: "Agent can list files and folders in any Tresor.",
				Parameters:  json.RawMessage(`{"tresor":"*","prefix":"*","max_keys":"*"}`),
			},
			{
				ID:          "tpl_tresorit_download",
				ActionType:  "tresorit.download_file",
				Name:        "Download from any Tresor",
				Description: "Agent can download files from any Tresor.",
				Parameters:  json.RawMessage(`{"tresor":"*","key":"*"}`),
			},
			{
				ID:          "tpl_tresorit_upload",
				ActionType:  "tresorit.upload_file",
				Name:        "Upload to any Tresor",
				Description: "Agent can upload files to any Tresor.",
				Parameters:  json.RawMessage(`{"tresor":"*","key":"*","content":"*"}`),
			},
		},
	}
}
