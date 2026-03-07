package asana

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *AsanaConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "asana",
		Name:        "Asana",
		Description: "Asana integration for project and task management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "asana.create_task",
				Name:        "Create Task",
				Description: "Create a new task in a project",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_id", "name"],
					"properties": {
						"project_id": {
							"type": "string",
							"description": "Project GID to create the task in"
						},
						"name": {
							"type": "string",
							"description": "Task name"
						},
						"notes": {
							"type": "string",
							"description": "Task description (supports rich text)"
						},
						"assignee": {
							"type": "string",
							"description": "Assignee user GID or email"
						},
						"due_on": {
							"type": "string",
							"description": "Due date (YYYY-MM-DD)"
						},
						"due_at": {
							"type": "string",
							"description": "Due date and time (ISO 8601)"
						},
						"tags": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Tag GIDs to apply"
						},
						"custom_fields": {
							"type": "object",
							"description": "Custom field GID to value mapping"
						}
					}
				}`)),
			},
			{
				ActionType:  "asana.update_task",
				Name:        "Update Task",
				Description: "Update fields on an existing task",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["task_id"],
					"properties": {
						"task_id": {
							"type": "string",
							"description": "Task GID to update"
						},
						"name": {
							"type": "string",
							"description": "Updated task name"
						},
						"notes": {
							"type": "string",
							"description": "Updated description"
						},
						"assignee": {
							"type": "string",
							"description": "Assignee user GID or email"
						},
						"due_on": {
							"type": "string",
							"description": "Due date (YYYY-MM-DD)"
						},
						"due_at": {
							"type": "string",
							"description": "Due date and time (ISO 8601)"
						},
						"completed": {
							"type": "boolean",
							"description": "Whether the task is completed"
						},
						"custom_fields": {
							"type": "object",
							"description": "Custom field GID to value mapping"
						}
					}
				}`)),
			},
			{
				ActionType:  "asana.add_comment",
				Name:        "Add Comment",
				Description: "Add a comment (story) to a task",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["task_id"],
					"properties": {
						"task_id": {
							"type": "string",
							"description": "Task GID to comment on"
						},
						"text": {
							"type": "string",
							"description": "Plain text comment"
						},
						"html_text": {
							"type": "string",
							"description": "Rich text comment (HTML)"
						}
					}
				}`)),
			},
			{
				ActionType:  "asana.complete_task",
				Name:        "Complete Task",
				Description: "Mark a task as complete",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["task_id"],
					"properties": {
						"task_id": {
							"type": "string",
							"description": "Task GID to complete"
						}
					}
				}`)),
			},
			{
				ActionType:  "asana.create_subtask",
				Name:        "Create Subtask",
				Description: "Create a subtask under an existing task",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["parent_task_id", "name"],
					"properties": {
						"parent_task_id": {
							"type": "string",
							"description": "Parent task GID"
						},
						"name": {
							"type": "string",
							"description": "Subtask name"
						},
						"notes": {
							"type": "string",
							"description": "Subtask description"
						},
						"assignee": {
							"type": "string",
							"description": "Assignee user GID or email"
						},
						"due_on": {
							"type": "string",
							"description": "Due date (YYYY-MM-DD)"
						}
					}
				}`)),
			},
			{
				ActionType:  "asana.search_tasks",
				Name:        "Search Tasks",
				Description: "Search and filter tasks across projects",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["workspace_id"],
					"properties": {
						"workspace_id": {
							"type": "string",
							"description": "Workspace GID to search in"
						},
						"text": {
							"type": "string",
							"description": "Full-text search query"
						},
						"assignee": {
							"type": "string",
							"description": "Filter by assignee GID or email"
						},
						"projects": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Filter by project GIDs"
						},
						"completed": {
							"type": "boolean",
							"description": "Filter by completion status"
						},
						"due_on_before": {
							"type": "string",
							"description": "Filter tasks due before this date (YYYY-MM-DD)"
						},
						"due_on_after": {
							"type": "string",
							"description": "Filter tasks due after this date (YYYY-MM-DD)"
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of results (default 20)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "asana", AuthType: "api_key", InstructionsURL: "https://developers.asana.com/docs/personal-access-token"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_asana_create_task_project",
				ActionType:  "asana.create_task",
				Name:        "Create tasks in a project",
				Description: "Agent can create tasks in a specific project with any name, description, and assignee.",
				Parameters:  json.RawMessage(`{"project_id":"*","name":"*","notes":"*","assignee":"*","due_on":"*"}`),
			},
			{
				ID:          "tpl_asana_update_task",
				ActionType:  "asana.update_task",
				Name:        "Update any task",
				Description: "Agent can update any task's fields.",
				Parameters:  json.RawMessage(`{"task_id":"*","name":"*","notes":"*","assignee":"*","due_on":"*","completed":"*"}`),
			},
			{
				ID:          "tpl_asana_add_comment",
				ActionType:  "asana.add_comment",
				Name:        "Comment on any task",
				Description: "Agent can add comments to any task.",
				Parameters:  json.RawMessage(`{"task_id":"*","text":"*"}`),
			},
			{
				ID:          "tpl_asana_complete_task",
				ActionType:  "asana.complete_task",
				Name:        "Complete any task",
				Description: "Agent can mark any task as complete.",
				Parameters:  json.RawMessage(`{"task_id":"*"}`),
			},
			{
				ID:          "tpl_asana_create_subtask",
				ActionType:  "asana.create_subtask",
				Name:        "Create subtasks",
				Description: "Agent can create subtasks under any task.",
				Parameters:  json.RawMessage(`{"parent_task_id":"*","name":"*","notes":"*","assignee":"*","due_on":"*"}`),
			},
			{
				ID:          "tpl_asana_search_tasks",
				ActionType:  "asana.search_tasks",
				Name:        "Search my incomplete tasks",
				Description: "Agent can search for incomplete tasks in a workspace.",
				Parameters:  json.RawMessage(`{"workspace_id":"*","text":"*","assignee":"*","completed":false,"limit":20}`),
			},
		},
	}
}
