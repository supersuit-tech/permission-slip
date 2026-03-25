package jira

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

func (c *JiraConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "jira",
		Name:        "Jira",
		Description: "Jira Cloud integration for issue tracking and project management",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "jira.create_issue",
				Name:        "Create Issue",
				Description: "Create a new issue in a Jira project",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_key", "issue_type", "summary"],
					"properties": {
						"project_key": {
							"type": "string",
							"description": "Project key (e.g. PROJ)",
							"x-ui": {
								"label": "Project Key",
								"placeholder": "PROJ",
								"help_text": "Short project identifier — visible in issue keys like PROJ-123"
							}
						},
						"issue_type": {
							"type": "string",
							"description": "Issue type (e.g. Bug, Story, Task)",
							"x-ui": {
								"label": "Issue Type",
								"placeholder": "Task",
								"help_text": "Use jira.list_issue_types to see available types"
							}
						},
						"summary": {
							"type": "string",
							"description": "Issue summary/title",
							"x-ui": {
								"label": "Summary",
								"placeholder": "Brief description"
							}
						},
						"description": {
							"type": "string",
							"description": "Issue description (plain text, converted to ADF)",
							"x-ui": {
								"label": "Description",
								"widget": "textarea"
							}
						},
						"assignee": {
							"type": "string",
							"description": "Atlassian account ID of the assignee",
							"x-ui": {
								"label": "Assignee",
								"placeholder": "5b10ac8d82e05b22cc7d4ef5",
								"help_text": "Atlassian account ID — find via user profile or People page"
							}
						},
						"priority": {
							"type": "string",
							"description": "Priority name (e.g. High, Medium, Low)",
							"x-ui": {
								"label": "Priority",
								"placeholder": "Medium"
							}
						},
						"labels": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Labels to apply to the issue",
							"x-ui": {
								"label": "Labels",
								"help_text": "One or more labels to categorize the issue"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.update_issue",
				Name:        "Update Issue",
				Description: "Update fields on an existing Jira issue",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_key"],
					"properties": {
						"issue_key": {
							"type": "string",
							"description": "Issue key (e.g. PROJ-123)",
							"x-ui": {
								"label": "Issue Key",
								"placeholder": "PROJ-123"
							}
						},
						"summary": {
							"type": "string",
							"description": "Updated summary/title",
							"x-ui": {
								"label": "Summary",
								"placeholder": "Brief description"
							}
						},
						"description": {
							"type": "string",
							"description": "Updated description (plain text, converted to ADF)",
							"x-ui": {
								"label": "Description",
								"widget": "textarea"
							}
						},
						"assignee": {
							"type": "string",
							"description": "Atlassian account ID of the assignee",
							"x-ui": {
								"label": "Assignee",
								"placeholder": "5b10ac8d82e05b22cc7d4ef5",
								"help_text": "Atlassian account ID — find via user profile or People page"
							}
						},
						"priority": {
							"type": "string",
							"description": "Priority name (e.g. High, Medium, Low)",
							"x-ui": {
								"label": "Priority",
								"placeholder": "Medium"
							}
						},
						"labels": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Labels to set on the issue",
							"x-ui": {
								"label": "Labels",
								"help_text": "One or more labels to categorize the issue"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.transition_issue",
				Name:        "Transition Issue",
				Description: "Move an issue through workflow states",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_key"],
					"properties": {
						"issue_key": {
							"type": "string",
							"description": "Issue key (e.g. PROJ-123)",
							"x-ui": {
								"label": "Issue Key",
								"placeholder": "PROJ-123"
							}
						},
						"transition_id": {
							"type": "string",
							"description": "Transition ID to apply",
							"x-ui": {
								"label": "Transition ID",
								"help_text": "Use jira.list_statuses to discover valid transitions"
							}
						},
						"transition_name": {
							"type": "string",
							"description": "Transition name (e.g. In Progress, Done). Looked up if transition_id is not provided.",
							"x-ui": {
								"label": "Transition Name",
								"placeholder": "In Progress"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.add_comment",
				Name:        "Add Comment",
				Description: "Add a comment to a Jira issue",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_key", "body"],
					"properties": {
						"issue_key": {
							"type": "string",
							"description": "Issue key (e.g. PROJ-123)",
							"x-ui": {
								"label": "Issue Key",
								"placeholder": "PROJ-123"
							}
						},
						"body": {
							"type": "string",
							"description": "Comment text (plain text, converted to ADF)",
							"x-ui": {
								"label": "Comment Body",
								"widget": "textarea"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.assign_issue",
				Name:        "Assign Issue",
				Description: "Assign an issue to a user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_key", "account_id"],
					"properties": {
						"issue_key": {
							"type": "string",
							"description": "Issue key (e.g. PROJ-123)",
							"x-ui": {
								"label": "Issue Key",
								"placeholder": "PROJ-123"
							}
						},
						"account_id": {
							"type": "string",
							"description": "Atlassian account ID of the user",
							"x-ui": {
								"label": "Account ID",
								"placeholder": "5b10ac8d82e05b22cc7d4ef5",
								"help_text": "Atlassian account ID — find via user profile or People page"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.search",
				Name:        "Search Issues",
				Description: "Search issues using JQL queries",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["jql"],
					"properties": {
						"jql": {
							"type": "string",
							"description": "JQL query string",
							"x-ui": {
								"label": "JQL Query",
								"placeholder": "project = PROJ AND status = 'In Progress'",
								"help_text": "Jira Query Language — see https://support.atlassian.com/jira-service-management-cloud/docs/use-advanced-search-with-jira-query-language-jql/"
							}
						},
						"max_results": {
							"type": "integer",
							"default": 50,
							"description": "Maximum number of results to return (max 1000)",
							"x-ui": {
								"label": "Max Results",
								"help_text": "Number of issues to return, between 1 and 1000"
							}
						},
						"fields": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Fields to include in the response",
							"x-ui": {
								"label": "Fields",
								"help_text": "Jira field names to include (e.g. summary, status, assignee). Omit for all fields."
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.list_projects",
				Name:        "List Projects",
				Description: "List all accessible projects — use to discover project keys for creating issues",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:  "jira.get_issue",
				Name:        "Get Issue",
				Description: "Get a single issue by key — read before updating to see current state",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_key"],
					"properties": {
						"issue_key": {
							"type": "string",
							"description": "Issue key (e.g. PROJ-123)",
							"x-ui": {
								"label": "Issue Key",
								"placeholder": "PROJ-123"
							}
						},
						"fields": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Fields to include in the response",
							"x-ui": {
								"label": "Fields",
								"help_text": "Jira field names to include (e.g. summary, status, assignee). Omit for all fields."
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.delete_issue",
				Name:        "Delete Issue",
				Description: "Delete an issue by key",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["issue_key"],
					"properties": {
						"issue_key": {
							"type": "string",
							"description": "Issue key (e.g. PROJ-123)",
							"x-ui": {
								"label": "Issue Key",
								"placeholder": "PROJ-123"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.list_sprints",
				Name:        "List Sprints",
				Description: "List sprints in a Jira board",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["board_id"],
					"properties": {
						"board_id": {
							"type": "integer",
							"description": "Board ID to list sprints for",
							"x-ui": {
								"label": "Board ID",
								"help_text": "Visible in the board URL (e.g. /board/42)"
							}
						},
						"state": {
							"type": "string",
							"enum": ["future", "active", "closed"],
							"description": "Filter sprints by state",
							"x-ui": {
								"label": "State",
								"widget": "select"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.create_sprint",
				Name:        "Create Sprint",
				Description: "Create a new sprint in a board",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name", "board_id"],
					"properties": {
						"name": {
							"type": "string",
							"description": "Sprint name",
							"x-ui": {
								"label": "Sprint Name",
								"placeholder": "Sprint 1"
							}
						},
						"board_id": {
							"type": "integer",
							"description": "Board ID to create the sprint in",
							"x-ui": {
								"label": "Board ID",
								"help_text": "Visible in the board URL (e.g. /board/42)"
							}
						},
						"goal": {
							"type": "string",
							"description": "Sprint goal",
							"x-ui": {
								"label": "Sprint Goal",
								"widget": "textarea"
							}
						},
						"start_date": {
							"type": "string",
							"format": "date-time",
							"description": "Sprint start date (ISO 8601, e.g. 2024-01-15T09:00:00.000Z)",
							"x-ui": {
								"label": "Start Date",
								"widget": "datetime",
								"help_text": "ISO 8601 format (e.g. 2024-01-15T09:00:00.000Z)",
								"datetime_range_pair": "end_date",
								"datetime_range_role": "lower"
							}
						},
						"end_date": {
							"type": "string",
							"format": "date-time",
							"description": "Sprint end date (ISO 8601, e.g. 2024-01-29T09:00:00.000Z)",
							"x-ui": {
								"label": "End Date",
								"widget": "datetime",
								"help_text": "ISO 8601 format (e.g. 2024-01-29T09:00:00.000Z)",
								"datetime_range_pair": "start_date",
								"datetime_range_role": "upper"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.move_to_sprint",
				Name:        "Move to Sprint",
				Description: "Move issues to a sprint",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["sprint_id", "issues"],
					"properties": {
						"sprint_id": {
							"type": "integer",
							"description": "Sprint ID to move issues to",
							"x-ui": {
								"label": "Sprint ID",
								"help_text": "Use jira.list_sprints to find IDs"
							}
						},
						"issues": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Issue keys to move (e.g. PROJ-1, PROJ-2)",
							"x-ui": {
								"label": "Issues",
								"help_text": "One or more issue keys to move into the sprint"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.list_statuses",
				Name:        "List Statuses",
				Description: "List workflow statuses — use with transition_issue to discover valid transitions",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"project_key": {
							"type": "string",
							"description": "Filter statuses by project key (optional)",
							"x-ui": {
								"label": "Project Key",
								"placeholder": "PROJ",
								"help_text": "Short project identifier — visible in issue keys like PROJ-123"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "jira.list_issue_types",
				Name:        "List Issue Types",
				Description: "List issue types (Bug, Story, Task, etc.) — required for creating issues with valid types",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "jira",
				AuthType:      "oauth2",
				OAuthProvider: "atlassian",
				OAuthScopes: []string{
					"read:me",
					"read:jira-work",
					"write:jira-work",
					"offline_access",
				},
			},
			{
				Service:         "jira",
				AuthType:        "basic",
				InstructionsURL: "https://support.atlassian.com/atlassian-account/docs/manage-api-tokens-for-your-atlassian-account/",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_jira_create_issue_project",
				ActionType:  "jira.create_issue",
				Name:        "Create issues in a project",
				Description: "Agent can create issues in a specific project with any type, summary, and details.",
				Parameters:  json.RawMessage(`{"project_key":"YOUR_PROJECT","issue_type":"*","summary":"*","description":"*","assignee":"*","priority":"*","labels":"*"}`),
			},
			{
				ID:          "tpl_jira_create_issue_all",
				ActionType:  "jira.create_issue",
				Name:        "Create issues (all projects)",
				Description: "Agent can create issues in any project with all fields open.",
				Parameters:  json.RawMessage(`{"project_key":"*","issue_type":"*","summary":"*","description":"*","assignee":"*","priority":"*","labels":"*"}`),
			},
			{
				ID:          "tpl_jira_transition_issue",
				ActionType:  "jira.transition_issue",
				Name:        "Transition issues",
				Description: "Agent can move any issue through workflow states.",
				Parameters:  json.RawMessage(`{"issue_key":"*","transition_id":"*","transition_name":"*"}`),
			},
			{
				ID:          "tpl_jira_search_assigned",
				ActionType:  "jira.search",
				Name:        "Search issues assigned to me",
				Description: "Search for issues assigned to the current user.",
				Parameters:  json.RawMessage(`{"jql":"assignee = currentUser()","max_results":"*","fields":"*"}`),
			},
		},
	}
}
