package linear

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//
//go:embed logo.svg
var logoSVG string

func (c *LinearConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "linear",
		Name:        "Linear",
		Description: "Linear integration for issue tracking and project management",
		LogoSVG:     logoSVG,
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
							"description": "The team ID to create the issue in",
							"x-ui": {"label": "Team", "help_text": "Team ID — use linear.list_teams to find team IDs"}
						},
						"title": {
							"type": "string",
							"description": "Issue title",
							"x-ui": {"label": "Title", "placeholder": "Enter issue title"}
						},
						"description": {
							"type": "string",
							"description": "Issue description (markdown)",
							"x-ui": {"label": "Description", "widget": "textarea"}
						},
						"assignee_id": {
							"type": "string",
							"description": "User ID to assign the issue to",
							"x-ui": {"label": "Assignee", "help_text": "User ID — find in team member settings"}
						},
						"priority": {
							"type": "integer",
							"minimum": 0,
							"maximum": 4,
							"description": "Priority: 0=none, 1=urgent, 2=high, 3=medium, 4=low",
							"x-ui": {"label": "Priority", "help_text": "0 = No priority, 1 = Urgent, 2 = High, 3 = Medium, 4 = Low", "widget": "select"}
						},
						"state_id": {
							"type": "string",
							"description": "Workflow state ID",
							"x-ui": {"label": "State", "help_text": "Workflow state ID — use linear.list_states or linear.search_issues to find state IDs"}
						},
						"label_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Label IDs to apply",
							"x-ui": {"label": "Labels", "help_text": "Use linear.list_labels to find label IDs"}
						},
						"project_id": {
							"type": "string",
							"description": "Project ID to associate with",
							"x-ui": {"label": "Project", "help_text": "Project ID — use linear.list_projects to find IDs"}
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
							"description": "The issue ID to update",
							"x-ui": {"label": "Issue", "help_text": "Linear issue ID (UUID format)"}
						},
						"title": {
							"type": "string",
							"description": "New issue title",
							"x-ui": {"label": "Title", "placeholder": "Enter new issue title"}
						},
						"description": {
							"type": "string",
							"description": "New issue description (markdown)",
							"x-ui": {"label": "Description", "widget": "textarea"}
						},
						"assignee_id": {
							"type": "string",
							"description": "User ID to assign the issue to",
							"x-ui": {"label": "Assignee", "help_text": "User ID — find in team member settings"}
						},
						"priority": {
							"type": "integer",
							"minimum": 0,
							"maximum": 4,
							"description": "Priority: 0=none, 1=urgent, 2=high, 3=medium, 4=low",
							"x-ui": {"label": "Priority", "help_text": "0 = No priority, 1 = Urgent, 2 = High, 3 = Medium, 4 = Low", "widget": "select"}
						},
						"state_id": {
							"type": "string",
							"description": "Workflow state ID",
							"x-ui": {"label": "State", "help_text": "Workflow state ID — use linear.list_states or linear.search_issues to find state IDs"}
						},
						"label_ids": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Label IDs to apply",
							"x-ui": {"label": "Labels", "help_text": "Use linear.list_labels to find label IDs"}
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
							"description": "The issue ID to comment on",
							"x-ui": {"label": "Issue", "help_text": "Linear issue ID (UUID format)"}
						},
						"body": {
							"type": "string",
							"description": "Comment body (markdown)",
							"x-ui": {"label": "Comment", "widget": "textarea"}
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
							"description": "Team IDs to associate with the project",
							"x-ui": {"label": "Teams", "help_text": "Team IDs — use linear.list_teams to find team IDs"}
						},
						"name": {
							"type": "string",
							"description": "Project name",
							"x-ui": {"label": "Name", "placeholder": "Enter project name"}
						},
						"description": {
							"type": "string",
							"description": "Project description",
							"x-ui": {"label": "Description", "widget": "textarea"}
						},
						"state": {
							"type": "string",
							"enum": ["planned", "started", "paused", "completed", "cancelled"],
							"description": "Project state",
							"x-ui": {"label": "State", "widget": "select", "help_text": "Valid values: planned, started, paused, completed, cancelled"}
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
							"description": "Search query (full-text across titles, descriptions, and comments when no filters are specified; matched against titles when filters are used)",
							"x-ui": {"label": "Query", "placeholder": "Search issues..."}
						},
						"team_id": {
							"type": "string",
							"description": "Filter by team ID",
							"x-ui": {"label": "Team", "help_text": "Team ID — use linear.list_teams to find team IDs"}
						},
						"assignee_id": {
							"type": "string",
							"description": "Filter by assignee user ID",
							"x-ui": {"label": "Assignee", "help_text": "User ID — find in team member settings"}
						},
						"state": {
							"type": "string",
							"description": "Filter by workflow state name",
							"x-ui": {"label": "State", "placeholder": "e.g. In Progress, Done", "help_text": "Workflow state name (not ID) — e.g. Backlog, Todo, In Progress, Done"}
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 100,
							"default": 50,
							"description": "Maximum number of results (default 50, max 100)",
							"x-ui": {"label": "Max results"}
						}
					}
				}`)),
			},
			{
				ActionType:  "linear.list_teams",
				Name:        "List Teams",
				Description: "List all teams — use to discover team IDs for creating issues",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"properties": {}
			}`)),
			},
			{
				ActionType:  "linear.get_issue",
				Name:        "Get Issue",
				Description: "Get a single issue by ID — returns full details including state, assignee, labels, and cycle",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["issue_id"],
				"properties": {
					"issue_id": {
						"type": "string",
						"description": "The issue ID to retrieve",
						"x-ui": {"label": "Issue", "help_text": "Linear issue ID (UUID format)"}
					}
				}
			}`)),
			},
			{
				ActionType:  "linear.assign_issue",
				Name:        "Assign Issue",
				Description: "Assign or unassign an issue — pass empty assignee_id to unassign",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["issue_id"],
				"properties": {
					"issue_id": {
						"type": "string",
						"description": "The issue ID to assign",
						"x-ui": {"label": "Issue", "help_text": "Linear issue ID (UUID format)"}
					},
					"assignee_id": {
						"type": "string",
						"description": "User ID to assign (empty to unassign)",
						"x-ui": {"label": "Assignee", "help_text": "User ID — find in team member settings"}
					}
				}
			}`)),
			},
			{
				ActionType:  "linear.change_state",
				Name:        "Change State",
				Description: "Change an issue's workflow state",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["issue_id", "state_id"],
				"properties": {
					"issue_id": {
						"type": "string",
						"description": "The issue ID to update",
						"x-ui": {"label": "Issue", "help_text": "Linear issue ID (UUID format)"}
					},
					"state_id": {
						"type": "string",
						"description": "The target workflow state ID",
						"x-ui": {"label": "State", "help_text": "Workflow state ID — use linear.list_states or linear.search_issues to find state IDs"}
					}
				}
			}`)),
			},
			{
				ActionType:  "linear.list_labels",
				Name:        "List Labels",
				Description: "List available labels — use to discover label IDs for add_label",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"properties": {
					"team_id": {
						"type": "string",
						"description": "Filter labels by team ID (optional)",
						"x-ui": {"label": "Team", "help_text": "Team ID — use linear.list_teams to find team IDs"}
					}
				}
			}`)),
			},
			{
				ActionType:  "linear.add_label",
				Name:        "Add Label",
				Description: "Add a label to an issue",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["issue_id", "label_id"],
				"properties": {
					"issue_id": {
						"type": "string",
						"description": "The issue ID to label",
						"x-ui": {"label": "Issue", "help_text": "Linear issue ID (UUID format)"}
					},
					"label_id": {
						"type": "string",
						"description": "The label ID to add",
						"x-ui": {"label": "Label", "help_text": "Use linear.list_labels to find label IDs"}
					}
				}
			}`)),
			},
			{
				ActionType:  "linear.list_cycles",
				Name:        "List Cycles",
				Description: "List cycles (sprints) for a team",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["team_id"],
				"properties": {
					"team_id": {
						"type": "string",
						"description": "Team ID to list cycles for",
						"x-ui": {"label": "Team", "help_text": "Team ID — use linear.list_teams to find team IDs"}
					}
				}
			}`)),
			},
		},
		// Two auth methods: OAuth (recommended) and API key (fallback).
		// Service names must be unique, so the OAuth entry uses "linear_oauth"
		// while the API key entry uses "linear". The credential resolver
		// tries OAuth first and falls back to the API key.
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "linear_oauth",
				AuthType:      "oauth2",
				OAuthProvider: "linear",
				OAuthScopes:   []string{"read", "write"},
			},
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
			{
				ID:          "tpl_linear_create_project",
				ActionType:  "linear.create_project",
				Name:        "Create projects in teams",
				Description: "Locks the teams and lets the agent choose project details.",
				Parameters:  json.RawMessage(`{"team_ids":"TEAM_IDS","name":"*","description":"*"}`),
			},
			{
				ID:          "tpl_linear_list_teams",
				ActionType:  "linear.list_teams",
				Name:        "List teams",
				Description: "Agent can discover available teams in the workspace.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_linear_get_issue",
				ActionType:  "linear.get_issue",
				Name:        "Get issue details",
				Description: "Agent can read any issue's full details.",
				Parameters:  json.RawMessage(`{"issue_id":"*"}`),
			},
			{
				ID:          "tpl_linear_list_labels",
				ActionType:  "linear.list_labels",
				Name:        "List labels for a team",
				Description: "Agent can list available labels, optionally filtered by team.",
				Parameters:  json.RawMessage(`{"team_id":"*"}`),
			},
			{
				ID:          "tpl_linear_list_cycles",
				ActionType:  "linear.list_cycles",
				Name:        "List cycles for a team",
				Description: "Agent can view sprint cycles for a specific team.",
				Parameters:  json.RawMessage(`{"team_id":"TEAM_ID"}`),
			},
		},
	}
}
