package jira

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *JiraConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "jira",
		Name:        "Jira",
		Description: "Jira Cloud integration for issue tracking and project management",
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
							"description": "Project key (e.g. PROJ)"
						},
						"issue_type": {
							"type": "string",
							"description": "Issue type (e.g. Bug, Story, Task)"
						},
						"summary": {
							"type": "string",
							"description": "Issue summary/title"
						},
						"description": {
							"type": "string",
							"description": "Issue description (plain text, converted to ADF)"
						},
						"assignee": {
							"type": "string",
							"description": "Atlassian account ID of the assignee"
						},
						"priority": {
							"type": "string",
							"description": "Priority name (e.g. High, Medium, Low)"
						},
						"labels": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Labels to apply to the issue"
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
							"description": "Issue key (e.g. PROJ-123)"
						},
						"summary": {
							"type": "string",
							"description": "Updated summary/title"
						},
						"description": {
							"type": "string",
							"description": "Updated description (plain text, converted to ADF)"
						},
						"assignee": {
							"type": "string",
							"description": "Atlassian account ID of the assignee"
						},
						"priority": {
							"type": "string",
							"description": "Priority name (e.g. High, Medium, Low)"
						},
						"labels": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Labels to set on the issue"
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
							"description": "Issue key (e.g. PROJ-123)"
						},
						"transition_id": {
							"type": "string",
							"description": "Transition ID to apply"
						},
						"transition_name": {
							"type": "string",
							"description": "Transition name (e.g. In Progress, Done). Looked up if transition_id is not provided."
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
							"description": "Issue key (e.g. PROJ-123)"
						},
						"body": {
							"type": "string",
							"description": "Comment text (plain text, converted to ADF)"
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
							"description": "Issue key (e.g. PROJ-123)"
						},
						"account_id": {
							"type": "string",
							"description": "Atlassian account ID of the user"
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
							"description": "JQL query string"
						},
						"max_results": {
							"type": "integer",
							"default": 50,
							"description": "Maximum number of results to return (max 1000)"
						},
						"fields": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Fields to include in the response"
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
