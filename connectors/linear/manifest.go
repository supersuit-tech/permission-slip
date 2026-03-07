package linear

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *LinearConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "linear",
		Name:        "Linear",
		Description: "Linear integration for issue tracking and project management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "linear.create_issue",
				Name:        "Create Issue",
				Description: "Create a new issue in a Linear team",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["team_id", "title"],
					"properties": {
						"team_id": {
							"type": "string",
							"description": "The team ID to create the issue in"
						},
						"title": {
							"type": "string",
							"description": "Issue title"
						},
						"description": {
							"type": "string",
							"description": "Issue description (markdown)"
						},
						"assignee_id": {
							"type": "string",
							"description": "User ID to assign the issue to"
						},
						"priority": {
							"type": "integer",
							"minimum": 0,
							"maximum": 4,
							"description": "Priority: 0=none, 1=urgent, 2=high, 3=medium, 4=low"
						},
						"state_id": {
							"type": "string",
							"description": "Workflow state ID"
						},
						"label_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Label IDs to apply"
						},
						"project_id": {
							"type": "string",
							"description": "Project ID to associate with"
						}
					}
				}`)),
			},
			{
				ActionType:  "linear.update_issue",
				Name:        "Update Issue",
				Description: "Update fields on an existing Linear issue",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_id"],
					"properties": {
						"issue_id": {
							"type": "string",
							"description": "The issue ID to update"
						},
						"title": {
							"type": "string",
							"description": "New issue title"
						},
						"description": {
							"type": "string",
							"description": "New issue description (markdown)"
						},
						"assignee_id": {
							"type": "string",
							"description": "User ID to assign the issue to"
						},
						"priority": {
							"type": "integer",
							"minimum": 0,
							"maximum": 4,
							"description": "Priority: 0=none, 1=urgent, 2=high, 3=medium, 4=low"
						},
						"state_id": {
							"type": "string",
							"description": "Workflow state ID"
						},
						"label_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Label IDs to apply"
						}
					}
				}`)),
			},
			{
				ActionType:  "linear.add_comment",
				Name:        "Add Comment",
				Description: "Add a comment to a Linear issue",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_id", "body"],
					"properties": {
						"issue_id": {
							"type": "string",
							"description": "The issue ID to comment on"
						},
						"body": {
							"type": "string",
							"description": "Comment body (markdown)"
						}
					}
				}`)),
			},
			{
				ActionType:  "linear.create_project",
				Name:        "Create Project",
				Description: "Create a new Linear project",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["team_ids", "name"],
					"properties": {
						"team_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Team IDs to associate with the project"
						},
						"name": {
							"type": "string",
							"description": "Project name"
						},
						"description": {
							"type": "string",
							"description": "Project description"
						},
						"state": {
							"type": "string",
							"enum": ["planned", "started", "paused", "completed", "cancelled"],
							"description": "Project state"
						}
					}
				}`)),
			},
			{
				ActionType:  "linear.search_issues",
				Name:        "Search Issues",
				Description: "Search Linear issues with full-text search or filtered queries",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Search query (matched against issue titles)"
						},
						"team_id": {
							"type": "string",
							"description": "Filter by team ID"
						},
						"assignee_id": {
							"type": "string",
							"description": "Filter by assignee user ID"
						},
						"state": {
							"type": "string",
							"description": "Filter by workflow state name"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 100,
							"default": 50,
							"description": "Maximum number of results (default 50, max 100)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "linear", AuthType: "api_key", InstructionsURL: "https://linear.app/docs/graphql/working-with-the-graphql-api#personal-api-keys"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_linear_create_issue_in_team",
				ActionType:  "linear.create_issue",
				Name:        "Create issues in a team",
				Description: "Locks the team and lets the agent choose issue details.",
				Parameters:  json.RawMessage(`{"team_id":"TEAM_ID","title":"*","description":"*"}`),
			},
			{
				ID:          "tpl_linear_search_my_issues",
				ActionType:  "linear.search_issues",
				Name:        "Search my assigned issues",
				Description: "Locks the assignee and lets the agent search freely.",
				Parameters:  json.RawMessage(`{"query":"*","assignee_id":"USER_ID"}`),
			},
			{
				ID:          "tpl_linear_add_comment",
				ActionType:  "linear.add_comment",
				Name:        "Comment on issues",
				Description: "Agent can add comments to any issue.",
				Parameters:  json.RawMessage(`{"issue_id":"*","body":"*"}`),
			},
			{
				ID:          "tpl_linear_update_issue",
				ActionType:  "linear.update_issue",
				Name:        "Update issues",
				Description: "Agent can update any issue's fields.",
				Parameters:  json.RawMessage(`{"issue_id":"*","title":"*","description":"*","priority":"*","state_id":"*"}`),
			},
		},
	}
}
