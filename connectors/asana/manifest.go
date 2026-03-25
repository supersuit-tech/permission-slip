package asana

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//go:embed logo.svg
var logoSVG string

func (c *AsanaConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "asana",
		Name:        "Asana",
		Description: "Asana integration for project and task management",
		LogoSVG:     logoSVG,
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
							"description": "Project GID to create the task in",
							"x-ui": {
								"label": "Project",
								"placeholder": "1234567890",
								"help_text": "Project GID — use asana.list_projects to find project IDs"
							}
						},
						"name": {
							"type": "string",
							"description": "Task name",
							"x-ui": {
								"label": "Name",
								"placeholder": "New task"
							}
						},
						"notes": {
							"type": "string",
							"description": "Task description (supports rich text)",
							"x-ui": {
								"label": "Description",
								"widget": "textarea"
							}
						},
						"assignee": {
							"type": "string",
							"description": "Assignee user GID or email",
							"x-ui": {
								"label": "Assignee",
								"placeholder": "me@example.com",
								"help_text": "User GID or email address"
							}
						},
						"due_on": {
							"type": "string",
							"format": "date",
							"description": "Due date (YYYY-MM-DD)",
							"x-ui": {
								"label": "Due date",
								"widget": "date",
								"help_text": "Date in YYYY-MM-DD format"
							}
						},
						"due_at": {
							"type": "string",
							"format": "date-time",
							"description": "Due date and time (ISO 8601)",
							"x-ui": {
								"label": "Due date & time",
								"widget": "datetime",
								"help_text": "Date and time in ISO 8601 format"
							}
						},
						"tags": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Tag GIDs to apply",
							"x-ui": {
								"label": "Tags",
								"help_text": "Tag GIDs — find via Asana tag settings"
							}
						},
						"custom_fields": {
							"type": "object",
							"description": "Custom field GID to value mapping",
							"x-ui": {
								"label": "Custom fields",
								"help_text": "Map of custom field GIDs to values"
							}
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
							"description": "Task GID to update",
							"x-ui": {
								"label": "Task",
								"placeholder": "1234567890",
								"help_text": "Task GID — visible in the task URL"
							}
						},
						"name": {
							"type": "string",
							"description": "Updated task name",
							"x-ui": {
								"label": "Name",
								"placeholder": "New task"
							}
						},
						"notes": {
							"type": "string",
							"description": "Updated description",
							"x-ui": {
								"label": "Description",
								"widget": "textarea"
							}
						},
						"assignee": {
							"type": "string",
							"description": "Assignee user GID or email",
							"x-ui": {
								"label": "Assignee",
								"placeholder": "me@example.com",
								"help_text": "User GID or email address"
							}
						},
						"due_on": {
							"type": "string",
							"format": "date",
							"description": "Due date (YYYY-MM-DD)",
							"x-ui": {
								"label": "Due date",
								"widget": "date",
								"help_text": "Date in YYYY-MM-DD format"
							}
						},
						"due_at": {
							"type": "string",
							"format": "date-time",
							"description": "Due date and time (ISO 8601)",
							"x-ui": {
								"label": "Due date & time",
								"widget": "datetime",
								"help_text": "Date and time in ISO 8601 format"
							}
						},
						"completed": {
							"type": "boolean",
							"description": "Whether the task is completed",
							"x-ui": {
								"label": "Completed",
								"widget": "toggle"
							}
						},
						"custom_fields": {
							"type": "object",
							"description": "Custom field GID to value mapping",
							"x-ui": {
								"label": "Custom fields",
								"help_text": "Map of custom field GIDs to values"
							}
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
							"description": "Task GID to comment on",
							"x-ui": {
								"label": "Task",
								"placeholder": "1234567890",
								"help_text": "Task GID — visible in the task URL"
							}
						},
						"text": {
							"type": "string",
							"description": "Plain text comment",
							"x-ui": {
								"label": "Comment",
								"widget": "textarea"
							}
						},
						"html_text": {
							"type": "string",
							"description": "Rich text comment (HTML)",
							"x-ui": {
								"label": "Rich text comment",
								"widget": "textarea"
							}
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
							"description": "Task GID to complete",
							"x-ui": {
								"label": "Task",
								"placeholder": "1234567890",
								"help_text": "Task GID — visible in the task URL"
							}
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
							"description": "Parent task GID",
							"x-ui": {
								"label": "Parent task",
								"placeholder": "1234567890",
								"help_text": "Parent task GID — visible in the task URL"
							}
						},
						"name": {
							"type": "string",
							"description": "Subtask name",
							"x-ui": {
								"label": "Name",
								"placeholder": "New task"
							}
						},
						"notes": {
							"type": "string",
							"description": "Subtask description",
							"x-ui": {
								"label": "Description",
								"widget": "textarea"
							}
						},
						"assignee": {
							"type": "string",
							"description": "Assignee user GID or email",
							"x-ui": {
								"label": "Assignee",
								"placeholder": "me@example.com",
								"help_text": "User GID or email address"
							}
						},
						"due_on": {
							"type": "string",
							"format": "date",
							"description": "Due date (YYYY-MM-DD)",
							"x-ui": {
								"label": "Due date",
								"widget": "date",
								"help_text": "Date in YYYY-MM-DD format"
							}
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
							"description": "Workspace GID to search in",
							"x-ui": {
								"label": "Workspace",
								"placeholder": "1234567890",
								"help_text": "Workspace GID — use asana.list_workspaces to find workspace IDs"
							}
						},
						"text": {
							"type": "string",
							"description": "Full-text search query",
							"x-ui": {
								"label": "Search query",
								"placeholder": "Search tasks..."
							}
						},
						"assignee": {
							"type": "string",
							"description": "Filter by assignee GID or email",
							"x-ui": {
								"label": "Assignee",
								"placeholder": "me@example.com",
								"help_text": "User GID or email address"
							}
						},
						"projects": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Filter by project GIDs",
							"x-ui": {
								"label": "Projects",
								"help_text": "Project GIDs to filter by"
							}
						},
						"completed": {
							"type": "boolean",
							"description": "Filter by completion status",
							"x-ui": {
								"label": "Completed",
								"widget": "toggle"
							}
						},
						"due_on_before": {
							"type": "string",
							"format": "date",
							"description": "Filter tasks due before this date (YYYY-MM-DD)",
							"x-ui": {
								"label": "Due before",
								"widget": "date",
								"help_text": "Date in YYYY-MM-DD format",
								"datetime_range_pair": "due_on_after",
								"datetime_range_role": "upper"
							}
						},
						"due_on_after": {
							"type": "string",
							"format": "date",
							"description": "Filter tasks due after this date (YYYY-MM-DD)",
							"x-ui": {
								"label": "Due after",
								"widget": "date",
								"help_text": "Date in YYYY-MM-DD format",
								"datetime_range_pair": "due_on_before",
								"datetime_range_role": "lower"
							}
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of results (default 20)",
							"x-ui": {
								"label": "Max results"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "asana.list_workspaces",
				Name:        "List Workspaces",
				Description: "List all workspaces accessible to the authenticated user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:  "asana.list_projects",
				Name:        "List Projects",
				Description: "List projects in a workspace",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["workspace_id"],
					"properties": {
						"workspace_id": {
							"type": "string",
							"description": "Workspace GID to list projects for",
							"x-ui": {
								"label": "Workspace",
								"placeholder": "1234567890",
								"help_text": "Workspace GID — use asana.list_workspaces to find workspace IDs"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "asana.create_project",
				Name:        "Create Project",
				Description: "Create a new project in a workspace",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["workspace_id", "name"],
					"properties": {
						"workspace_id": {
							"type": "string",
							"description": "Workspace GID to create the project in",
							"x-ui": {
								"label": "Workspace",
								"placeholder": "1234567890",
								"help_text": "Workspace GID — use asana.list_workspaces to find workspace IDs"
							}
						},
						"name": {
							"type": "string",
							"description": "Project name",
							"x-ui": {
								"label": "Name",
								"placeholder": "My Project"
							}
						},
						"notes": {
							"type": "string",
							"description": "Project description",
							"x-ui": {
								"label": "Description",
								"widget": "textarea"
							}
						},
						"color": {
							"type": "string",
							"description": "Project color (e.g. light-green, light-red)",
							"x-ui": {
								"label": "Color",
								"placeholder": "light-green"
							}
						},
						"privacy": {
							"type": "string",
							"enum": ["public_to_workspace", "private"],
							"description": "Privacy setting",
							"x-ui": {
								"label": "Privacy",
								"widget": "select"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "asana.delete_task",
				Name:        "Delete Task",
				Description: "Permanently delete a task",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["task_id"],
					"properties": {
						"task_id": {
							"type": "string",
							"description": "Task GID to delete",
							"x-ui": {
								"label": "Task",
								"placeholder": "1234567890",
								"help_text": "Task GID — visible in the task URL"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "asana.list_sections",
				Name:        "List Sections",
				Description: "List sections in a project",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_id"],
					"properties": {
						"project_id": {
							"type": "string",
							"description": "Project GID to list sections for",
							"x-ui": {
								"label": "Project",
								"placeholder": "1234567890",
								"help_text": "Project GID — use asana.list_projects to find project IDs"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "asana.create_section",
				Name:        "Create Section",
				Description: "Create a new section in a project",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["project_id", "name"],
					"properties": {
						"project_id": {
							"type": "string",
							"description": "Project GID to create the section in",
							"x-ui": {
								"label": "Project",
								"placeholder": "1234567890",
								"help_text": "Project GID — use asana.list_projects to find project IDs"
							}
						},
						"name": {
							"type": "string",
							"description": "Section name",
							"x-ui": {
								"label": "Name",
								"placeholder": "New task"
							}
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
			{
				ID:          "tpl_asana_list_workspaces",
				ActionType:  "asana.list_workspaces",
				Name:        "List workspaces",
				Description: "Agent can list all workspaces accessible to the user.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_asana_list_projects",
				ActionType:  "asana.list_projects",
				Name:        "List projects in a workspace",
				Description: "Agent can list all projects in a workspace.",
				Parameters:  json.RawMessage(`{"workspace_id":"*"}`),
			},
			{
				ID:          "tpl_asana_create_project",
				ActionType:  "asana.create_project",
				Name:        "Create a project",
				Description: "Agent can create new projects in a workspace.",
				Parameters:  json.RawMessage(`{"workspace_id":"*","name":"*","notes":"*"}`),
			},
			{
				ID:          "tpl_asana_delete_task",
				ActionType:  "asana.delete_task",
				Name:        "Delete any task",
				Description: "Agent can permanently delete any task.",
				Parameters:  json.RawMessage(`{"task_id":"*"}`),
			},
			{
				ID:          "tpl_asana_list_sections",
				ActionType:  "asana.list_sections",
				Name:        "List sections in a project",
				Description: "Agent can list all sections in a project.",
				Parameters:  json.RawMessage(`{"project_id":"*"}`),
			},
			{
				ID:          "tpl_asana_create_section",
				ActionType:  "asana.create_section",
				Name:        "Create a section",
				Description: "Agent can create sections in any project.",
				Parameters:  json.RawMessage(`{"project_id":"*","name":"*"}`),
			},
		},
	}
}
