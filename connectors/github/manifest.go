package github

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
//
//go:embed logo.svg
var logoSVG string

// neverExpire is the standing-approval spec applied to templates that opt in
// to a never-expiring auto-approval when the template is applied. Using the
// shared value keeps each template's declaration to a single line and makes
// the opt-in explicit rather than relying on a cross-cutting default.
var neverExpire = &connectors.ManifestStandingApproval{}

func (c *GitHubConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "github",
		Name:        "GitHub",
		Description: "GitHub integration for repository management",
		Status:      "early_preview",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:      "github.create_issue",
				Name:            "Create Issue",
				Description:     "Create a new issue in a repository",
				RiskLevel:       "low",
				DisplayTemplate: "Create issue {{title}} in {{owner}}/{{repo}}",
				Preview: &connectors.ActionPreview{
					Layout: "record",
					Fields: map[string]string{"title": "title", "subtitle": "repo"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "title"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"title": {
							"type": "string",
							"description": "Issue title",
							"x-ui": {"label": "Title", "placeholder": "Bug: something is broken"}
						},
						"body": {
							"type": "string",
							"description": "Issue body (Markdown supported)",
							"x-ui": {"widget": "textarea", "label": "Body"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.merge_pr",
				Name:            "Merge Pull Request",
				Description:     "Merge an open pull request",
				RiskLevel:       "medium",
				DisplayTemplate: "Merge PR #{{pull_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "pull_number"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"pull_number": {
							"type": "integer",
							"description": "Pull request number",
							"x-ui": {"label": "PR number"}
						},
						"merge_method": {
							"type": "string",
							"enum": ["merge", "squash", "rebase"],
							"default": "merge",
							"description": "Merge strategy to use",
							"x-ui": {"widget": "select", "label": "Merge method"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.create_pr",
				Name:            "Create Pull Request",
				Description:     "Create a pull request from a branch",
				RiskLevel:       "medium",
				DisplayTemplate: "Create PR {{title}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "title", "head", "base"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"title": {
							"type": "string",
							"description": "Pull request title",
							"x-ui": {"label": "Title", "placeholder": "Add new feature"}
						},
						"body": {
							"type": "string",
							"description": "Pull request body (Markdown supported)",
							"x-ui": {"widget": "textarea", "label": "Body"}
						},
						"head": {
							"type": "string",
							"description": "Branch containing the changes",
							"x-ui": {"label": "Head branch", "placeholder": "feature-branch"}
						},
						"base": {
							"type": "string",
							"description": "Branch to merge into",
							"x-ui": {"label": "Base branch", "placeholder": "main"}
						},
						"draft": {
							"type": "boolean",
							"default": false,
							"description": "Whether to create the PR as a draft",
							"x-ui": {"widget": "toggle", "label": "Draft"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.add_reviewer",
				Name:            "Add Reviewer",
				Description:     "Request reviews on a pull request",
				RiskLevel:       "low",
				DisplayTemplate: "Request review on PR #{{pull_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "pull_number", "reviewers"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"pull_number": {
							"type": "integer",
							"description": "Pull request number",
							"x-ui": {"label": "PR number"}
						},
						"reviewers": {
							"type": "array",
							"items": {"type": "string"},
							"description": "GitHub usernames to request reviews from",
							"x-ui": {"label": "Reviewers", "help_text": "GitHub usernames"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.create_release",
				Name:            "Create Release",
				Description:     "Create a tagged release in a repository",
				RiskLevel:       "medium",
				DisplayTemplate: "Create release {{tag_name}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "tag_name"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"tag_name": {
							"type": "string",
							"description": "The name of the tag for this release",
							"x-ui": {"label": "Tag", "placeholder": "v1.0.0"}
						},
						"name": {
							"type": "string",
							"description": "The name of the release",
							"x-ui": {"label": "Name", "placeholder": "v1.0.0"}
						},
						"body": {
							"type": "string",
							"description": "Release notes (Markdown supported)",
							"x-ui": {"widget": "textarea", "label": "Body"}
						},
						"draft": {
							"type": "boolean",
							"default": false,
							"description": "Whether to create as a draft release",
							"x-ui": {"widget": "toggle", "label": "Draft"}
						},
						"prerelease": {
							"type": "boolean",
							"default": false,
							"description": "Whether to mark as a pre-release",
							"x-ui": {"widget": "toggle", "label": "Pre-release"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.close_issue",
				Name:            "Close Issue",
				Description:     "Close an issue with an optional comment",
				RiskLevel:       "medium",
				DisplayTemplate: "Close issue #{{issue_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "issue_number"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"issue_number": {
							"type": "integer",
							"description": "Issue number to close",
							"x-ui": {"label": "Issue number"}
						},
						"state_reason": {
							"type": "string",
							"enum": ["completed", "not_planned"],
							"default": "completed",
							"description": "Reason for closing the issue",
							"x-ui": {"widget": "select", "label": "Close reason"}
						},
						"comment": {
							"type": "string",
							"description": "Optional comment to add before closing",
							"x-ui": {"widget": "textarea", "label": "Comment"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.add_label",
				Name:            "Add Label",
				Description:     "Add labels to an issue or pull request",
				RiskLevel:       "low",
				DisplayTemplate: "Add labels to #{{issue_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "issue_number", "labels"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"issue_number": {
							"type": "integer",
							"description": "Issue or pull request number",
							"x-ui": {"label": "Issue number"}
						},
						"labels": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Labels to add",
							"x-ui": {"label": "Labels"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.add_comment",
				Name:            "Add Comment",
				Description:     "Add a comment to an issue or pull request",
				RiskLevel:       "low",
				DisplayTemplate: "Comment on #{{issue_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "issue_number", "body"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"issue_number": {
							"type": "integer",
							"description": "Issue or pull request number",
							"x-ui": {"label": "Issue number"}
						},
						"body": {
							"type": "string",
							"description": "Comment body (Markdown supported)",
							"x-ui": {"widget": "textarea", "label": "Body"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.create_branch",
				Name:            "Create Branch",
				Description:     "Create a new branch from a ref",
				RiskLevel:       "low",
				DisplayTemplate: "Create branch {{branch_name}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "branch_name", "from_ref"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"branch_name": {
							"type": "string",
							"description": "Name for the new branch",
							"x-ui": {"label": "Branch name", "placeholder": "feature/my-feature"}
						},
						"from_ref": {
							"type": "string",
							"description": "Branch or ref to create from (e.g. \"main\", \"develop\", or \"tags/v1.0\")",
							"x-ui": {"label": "Source ref", "placeholder": "main"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.get_file_contents",
				Name:            "Get File Contents",
				Description:     "Read file contents from a repository",
				RiskLevel:       "low",
				DisplayTemplate: "Read {{path}} from {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "path"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"path": {
							"type": "string",
							"description": "File path within the repository (relative, no leading slash)",
							"x-ui": {"label": "File path", "placeholder": "src/index.ts"}
						},
						"ref": {
							"type": "string",
							"description": "Branch, tag, or commit SHA to read from (defaults to default branch)",
							"x-ui": {"label": "Branch/tag", "placeholder": "main"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.create_or_update_file",
				OperationType:   "edit",
				Name:            "Create or Update File",
				Description:     "Create or update a file in a repository via the Contents API",
				RiskLevel:       "medium",
				DisplayTemplate: "Write {{path}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "path", "message", "content"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"path": {
							"type": "string",
							"description": "File path within the repository (relative, no leading slash)",
							"x-ui": {"label": "File path", "placeholder": "src/index.ts"}
						},
						"message": {
							"type": "string",
							"description": "Commit message",
							"x-ui": {"label": "Commit message", "placeholder": "Update src/index.ts"}
						},
						"content": {
							"type": "string",
							"description": "New file content, base64-encoded",
							"x-ui": {"label": "File content (Base64)"}
						},
						"branch": {
							"type": "string",
							"description": "Branch to commit to (defaults to default branch)",
							"x-ui": {"label": "Branch", "placeholder": "main"}
						},
						"sha": {
							"type": "string",
							"description": "Blob SHA of the file being replaced (required when updating an existing file)",
							"x-ui": {"label": "File SHA", "help_text": "Required when updating — get from github.get_file_contents"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.list_repos",
				Name:            "List Repositories",
				Description:     "List repositories for the authenticated user or an organization",
				RiskLevel:       "low",
				DisplayTemplate: "List repositories",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"org": {
							"type": "string",
							"description": "Organization login to list repos for (omit to list the authenticated user's repos)",
							"x-ui": {"label": "Organization", "placeholder": "octocat", "help_text": "Omit to use authenticated user's repos"}
						},
						"type": {
							"type": "string",
							"enum": ["all", "public", "private", "forks", "sources", "member"],
							"description": "Repository type filter",
							"x-ui": {"widget": "select", "label": "Type"}
						},
						"visibility": {
							"type": "string",
							"enum": ["all", "public", "private"],
							"description": "Visibility filter — useful for org repos (all, public, private)",
							"x-ui": {"widget": "select", "label": "Visibility"}
						},
						"sort": {
							"type": "string",
							"enum": ["created", "updated", "pushed", "full_name"],
							"description": "Sort field",
							"x-ui": {"widget": "select", "label": "Sort"}
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Number of results per page (max 100)",
							"x-ui": {"label": "Results per page"}
						},
						"page": {
							"type": "integer",
							"default": 1,
							"description": "Page number for pagination (starts at 1)",
							"x-ui": {"label": "Page"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.get_repo",
				Name:            "Get Repository",
				Description:     "Get repository metadata",
				RiskLevel:       "low",
				DisplayTemplate: "Get repo {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.list_pull_requests",
				Name:            "List Pull Requests",
				Description:     "List pull requests for a repository with optional filtering",
				RiskLevel:       "low",
				DisplayTemplate: "List PRs in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"state": {
							"type": "string",
							"enum": ["open", "closed", "all"],
							"default": "open",
							"description": "State filter",
							"x-ui": {"widget": "select", "label": "State"}
						},
						"base": {
							"type": "string",
							"description": "Filter by base branch name",
							"x-ui": {"label": "Base branch", "placeholder": "main"}
						},
						"head": {
							"type": "string",
							"description": "Filter by head branch name (user:branch format)",
							"x-ui": {"label": "Head branch", "placeholder": "feature-branch"}
						},
						"sort": {
							"type": "string",
							"enum": ["created", "updated", "popularity", "long-running"],
							"description": "Sort field",
							"x-ui": {"widget": "select", "label": "Sort"}
						},
						"direction": {
							"type": "string",
							"enum": ["asc", "desc"],
							"description": "Sort direction",
							"x-ui": {"widget": "select", "label": "Direction"}
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Number of results per page (max 100)",
							"x-ui": {"label": "Results per page"}
						},
						"page": {
							"type": "integer",
							"default": 1,
							"description": "Page number for pagination (starts at 1)",
							"x-ui": {"label": "Page"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.trigger_workflow",
				Name:            "Trigger Workflow",
				Description:     "Trigger a GitHub Actions workflow dispatch event",
				RiskLevel:       "medium",
				DisplayTemplate: "Trigger workflow {{workflow_name}} ({{workflow_id}}) in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "workflow_id", "ref"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"workflow_id": {
							"type": "string",
							"description": "Workflow file name (e.g. \"deploy.yml\") or numeric workflow ID",
							"x-ui": {"label": "Workflow", "placeholder": "deploy.yml", "help_text": "Workflow file name or numeric ID"}
						},
						"ref": {
							"type": "string",
							"description": "Branch or tag to run the workflow on",
							"x-ui": {"label": "Branch/tag", "placeholder": "main"}
						},
						"inputs": {
							"type": "object",
							"description": "Input key-value pairs defined by the workflow's on.workflow_dispatch.inputs",
							"additionalProperties": true,
							"x-ui": {"label": "Workflow inputs", "help_text": "Key-value pairs matching the workflow's on.workflow_dispatch.inputs"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.list_workflow_runs",
				Name:            "List Workflow Runs",
				Description:     "List workflow runs for a repository with optional status, event, and actor filtering",
				RiskLevel:       "low",
				DisplayTemplate: "List workflow runs in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"workflow_id": {
							"type": "string",
							"description": "Workflow file name or ID to filter runs (omit for all workflows)",
							"x-ui": {"label": "Workflow", "placeholder": "deploy.yml", "help_text": "Workflow file name or numeric ID"}
						},
						"status": {
							"type": "string",
							"enum": ["completed", "action_required", "cancelled", "failure", "neutral", "skipped", "stale", "success", "timed_out", "in_progress", "queued", "requested", "waiting", "pending"],
							"description": "Filter by run status",
							"x-ui": {"widget": "select", "label": "Status"}
						},
						"branch": {
							"type": "string",
							"description": "Filter by branch name",
							"x-ui": {"label": "Branch", "placeholder": "main"}
						},
						"event": {
							"type": "string",
							"description": "Filter by triggering event (e.g. \"push\", \"pull_request\", \"workflow_dispatch\", \"schedule\")",
							"x-ui": {"label": "Event", "placeholder": "push"}
						},
						"actor": {
							"type": "string",
							"description": "Filter by the GitHub username that triggered the run",
							"x-ui": {"label": "Actor", "placeholder": "octocat"}
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Number of results per page (max 100)",
							"x-ui": {"label": "Results per page"}
						},
						"page": {
							"type": "integer",
							"default": 1,
							"description": "Page number for pagination (starts at 1)",
							"x-ui": {"label": "Page"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.create_webhook",
				Name:            "Create Webhook",
				Description:     "Create a repository webhook",
				RiskLevel:       "medium",
				DisplayTemplate: "Create webhook in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "url", "events"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat", "help_text": "GitHub username or organization"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"url": {
							"type": "string",
							"description": "Payload URL for the webhook",
							"x-ui": {"label": "Payload URL", "placeholder": "https://example.com/webhook"}
						},
						"events": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Events that trigger the webhook (e.g. [\"push\", \"pull_request\"])",
							"x-ui": {"label": "Events", "help_text": "e.g. push, pull_request, issues"}
						},
						"content_type": {
							"type": "string",
							"enum": ["json", "form"],
							"default": "json",
							"description": "Payload content type",
							"x-ui": {"widget": "select", "label": "Content type"}
						},
						"secret": {
							"type": "string",
							"description": "Secret used to sign webhook payloads",
							"x-ui": {"label": "Secret"}
						},
						"active": {
							"type": "boolean",
							"default": true,
							"description": "Whether the webhook is active",
							"x-ui": {"widget": "toggle", "label": "Active"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.search_code",
				Name:            "Search Code",
				Description:     "Search code across repositories",
				RiskLevel:       "low",
				DisplayTemplate: "Search code: {{q}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["q"],
					"properties": {
						"q": {
							"type": "string",
							"description": "Search query (supports GitHub code search qualifiers, e.g. \"myFunc language:go repo:owner/repo\")",
							"x-ui": {"label": "Search query", "placeholder": "myFunc language:go repo:owner/repo", "help_text": "Uses GitHub code search syntax — supports language:, repo:, path: qualifiers"}
						},
						"order": {
							"type": "string",
							"enum": ["asc", "desc"],
							"description": "Sort order for results",
							"x-ui": {"widget": "select", "label": "Order"}
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Number of results per page (max 100)",
							"x-ui": {"label": "Results per page"}
						},
						"page": {
							"type": "integer",
							"default": 1,
							"description": "Page number for pagination (starts at 1)",
							"x-ui": {"label": "Page"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.search_issues",
				Name:            "Search Issues",
				Description:     "Search issues and pull requests across repositories",
				RiskLevel:       "low",
				DisplayTemplate: "Search issues: {{q}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["q"],
					"properties": {
						"q": {
							"type": "string",
							"description": "Search query (supports GitHub issue search qualifiers, e.g. \"is:open label:bug repo:owner/repo\")",
							"x-ui": {"label": "Search query", "placeholder": "is:open label:bug repo:owner/repo", "help_text": "Uses GitHub issue search syntax — supports is:, label:, repo: qualifiers"}
						},
						"sort": {
							"type": "string",
							"enum": ["comments", "reactions", "reactions-+1", "reactions--1", "reactions-smile", "reactions-thinking_face", "reactions-heart", "reactions-tada", "interactions", "created", "updated"],
							"description": "Sort field",
							"x-ui": {"widget": "select", "label": "Sort"}
						},
						"order": {
							"type": "string",
							"enum": ["asc", "desc"],
							"description": "Sort order",
							"x-ui": {"widget": "select", "label": "Order"}
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Number of results per page (max 100)",
							"x-ui": {"label": "Results per page"}
						},
						"page": {
							"type": "integer",
							"default": 1,
							"description": "Page number for pagination (starts at 1)",
							"x-ui": {"label": "Page"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.delete_branch",
				Name:            "Delete Branch",
				Description:     "Delete a branch ref in a repository",
				RiskLevel:       "high",
				DisplayTemplate: "Delete branch {{branch_name}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "branch_name"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"branch_name": {
							"type": "string",
							"description": "Branch name to delete (not a full ref path)",
							"x-ui": {"label": "Branch", "placeholder": "feature/old-branch"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.delete_file",
				Name:            "Delete File",
				Description:     "Delete a file from a repository via the Contents API",
				RiskLevel:       "high",
				DisplayTemplate: "Delete {{path}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "path", "message", "sha"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"path": {
							"type": "string",
							"description": "File path within the repository (relative, no leading slash)",
							"x-ui": {"label": "File path", "placeholder": "src/old.ts"}
						},
						"message": {
							"type": "string",
							"description": "Commit message",
							"x-ui": {"label": "Commit message"}
						},
						"sha": {
							"type": "string",
							"description": "Blob SHA of the file to delete (from get_file_contents)",
							"x-ui": {"label": "File SHA"}
						},
						"branch": {
							"type": "string",
							"description": "Branch to commit on (defaults to default branch)",
							"x-ui": {"label": "Branch", "placeholder": "main"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.remove_label",
				Name:            "Remove Label",
				Description:     "Remove a single label from an issue or pull request",
				RiskLevel:       "low",
				DisplayTemplate: "Remove label {{name}} from #{{issue_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "issue_number", "name"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"issue_number": {
							"type": "integer",
							"description": "Issue or pull request number",
							"x-ui": {"label": "Issue number"}
						},
						"name": {
							"type": "string",
							"description": "Label name to remove",
							"x-ui": {"label": "Label", "placeholder": "bug"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.close_pr",
				Name:            "Close Pull Request",
				Description:     "Close an open pull request without merging",
				RiskLevel:       "medium",
				DisplayTemplate: "Close PR #{{pull_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "pull_number"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"pull_number": {
							"type": "integer",
							"description": "Pull request number",
							"x-ui": {"label": "PR number"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.update_issue",
				OperationType:   "edit",
				Name:            "Update Issue",
				Description:     "Update an issue title, body, assignees, or state",
				RiskLevel:       "low",
				DisplayTemplate: "Update issue #{{issue_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "issue_number"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"issue_number": {
							"type": "integer",
							"description": "Issue number",
							"x-ui": {"label": "Issue number"}
						},
						"title": {
							"type": "string",
							"description": "New title (omit to leave unchanged)",
							"x-ui": {"label": "Title"}
						},
						"body": {
							"type": "string",
							"description": "New body (omit to leave unchanged)",
							"x-ui": {"widget": "textarea", "label": "Body"}
						},
						"state": {
							"type": "string",
							"enum": ["open", "closed"],
							"description": "Issue state (omit to leave unchanged)",
							"x-ui": {"widget": "select", "label": "State"}
						},
						"assignees": {
							"type": "array",
							"items": {"type": "string"},
							"description": "GitHub usernames to set as assignees (omit to leave unchanged)",
							"x-ui": {"label": "Assignees"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.update_pr",
				OperationType:   "edit",
				Name:            "Update Pull Request",
				Description:     "Update a pull request title, body, base branch, or state",
				RiskLevel:       "low",
				DisplayTemplate: "Update PR #{{pull_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "pull_number"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"pull_number": {
							"type": "integer",
							"description": "Pull request number",
							"x-ui": {"label": "PR number"}
						},
						"title": {
							"type": "string",
							"description": "New title (omit to leave unchanged)",
							"x-ui": {"label": "Title"}
						},
						"body": {
							"type": "string",
							"description": "New body (omit to leave unchanged)",
							"x-ui": {"widget": "textarea", "label": "Body"}
						},
						"base": {
							"type": "string",
							"description": "New base branch (omit to leave unchanged)",
							"x-ui": {"label": "Base branch", "placeholder": "main"}
						},
						"state": {
							"type": "string",
							"enum": ["open", "closed"],
							"description": "PR state (omit to leave unchanged)",
							"x-ui": {"widget": "select", "label": "State"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.get_issue",
				Name:            "Get Issue",
				Description:     "Fetch a single issue by number",
				RiskLevel:       "low",
				DisplayTemplate: "Get issue #{{issue_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "issue_number"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"issue_number": {
							"type": "integer",
							"description": "Issue number",
							"x-ui": {"label": "Issue number"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.get_pr",
				Name:            "Get Pull Request",
				Description:     "Fetch a single pull request by number",
				RiskLevel:       "low",
				DisplayTemplate: "Get PR #{{pull_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "pull_number"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"pull_number": {
							"type": "integer",
							"description": "Pull request number",
							"x-ui": {"label": "PR number"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.list_issues",
				Name:            "List Issues",
				Description:     "List issues for a repository (pull requests excluded unless include_pull_requests is true)",
				RiskLevel:       "low",
				DisplayTemplate: "List issues in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"state": {
							"type": "string",
							"enum": ["open", "closed", "all"],
							"default": "open",
							"description": "Issue state filter",
							"x-ui": {"widget": "select", "label": "State"}
						},
						"labels": {
							"type": "string",
							"description": "Comma-separated list of label names",
							"x-ui": {"label": "Labels", "placeholder": "bug,enhancement"}
						},
						"sort": {
							"type": "string",
							"enum": ["created", "updated", "comments"],
							"description": "Sort field",
							"x-ui": {"widget": "select", "label": "Sort"}
						},
						"direction": {
							"type": "string",
							"enum": ["asc", "desc"],
							"description": "Sort direction",
							"x-ui": {"widget": "select", "label": "Direction"}
						},
						"since": {
							"type": "string",
							"format": "date-time",
							"description": "Only issues updated at or after this time (ISO 8601)",
							"x-ui": {"label": "Since", "widget": "datetime"}
						},
						"include_pull_requests": {
							"type": "boolean",
							"default": false,
							"description": "When true, include pull requests in results (GitHub API returns both by default)",
							"x-ui": {"widget": "toggle", "label": "Include PRs"}
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Results per page fetched from GitHub (max 100). When include_pull_requests is false, pull requests are filtered client-side, so the actual returned count may be lower than per_page.",
							"x-ui": {"label": "Results per page"}
						},
						"page": {
							"type": "integer",
							"default": 1,
							"description": "Page number",
							"x-ui": {"label": "Page"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.list_commits",
				Name:            "List Commits",
				Description:     "List commits for a repository",
				RiskLevel:       "low",
				DisplayTemplate: "List commits in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"sha": {
							"type": "string",
							"description": "SHA or branch to start listing from",
							"x-ui": {"label": "SHA / branch", "placeholder": "main"}
						},
						"path": {
							"type": "string",
							"description": "Only commits containing this file path",
							"x-ui": {"label": "Path", "placeholder": "src/"}
						},
						"author": {
							"type": "string",
							"description": "Git author email or name",
							"x-ui": {"label": "Author"}
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Results per page (max 100)",
							"x-ui": {"label": "Results per page"}
						},
						"page": {
							"type": "integer",
							"default": 1,
							"description": "Page number",
							"x-ui": {"label": "Page"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.remove_reviewer",
				Name:            "Remove Reviewer",
				Description:     "Remove requested reviewers from a pull request",
				RiskLevel:       "low",
				DisplayTemplate: "Remove reviewers from PR #{{pull_number}} in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "pull_number", "reviewers"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"pull_number": {
							"type": "integer",
							"description": "Pull request number",
							"x-ui": {"label": "PR number"}
						},
						"reviewers": {
							"type": "array",
							"items": {"type": "string"},
							"description": "GitHub usernames to remove from requested reviewers",
							"x-ui": {"label": "Reviewers"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.delete_webhook",
				Name:            "Delete Webhook",
				Description:     "Delete a repository webhook by numeric hook ID",
				RiskLevel:       "high",
				DisplayTemplate: "Delete webhook {{webhook_url}} (#{{hook_id}}) in {{owner}}/{{repo}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "hook_id"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)",
							"x-ui": {"label": "Owner", "placeholder": "octocat"}
						},
						"repo": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Repository", "placeholder": "my-repo"}
						},
						"hook_id": {
							"type": "integer",
							"description": "Numeric webhook ID from GitHub",
							"x-ui": {"label": "Hook ID"}
						}
					}
				}`)),
			},
			{
				ActionType:      "github.create_repo",
				Name:            "Create Repository",
				Description:     "Create a new repository for the authenticated user or an organization",
				RiskLevel:       "medium",
				DisplayTemplate: "Create repo {{name}}",
				Preview: &connectors.ActionPreview{
					Layout: "record",
					Fields: map[string]string{"title": "name", "subtitle": "org"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name"],
					"properties": {
						"name": {
							"type": "string",
							"description": "Repository name",
							"x-ui": {"label": "Name", "placeholder": "my-repo"}
						},
						"org": {
							"type": "string",
							"description": "Organization to create the repository in (omit to create under the authenticated user)",
							"x-ui": {"label": "Organization", "placeholder": "octocat", "help_text": "Omit to use authenticated user's repos"}
						},
						"description": {
							"type": "string",
							"description": "Repository description",
							"x-ui": {"label": "Description", "placeholder": "A short description of the repository"}
						},
						"private": {
							"type": "boolean",
							"default": false,
							"description": "Whether the repository is private",
							"x-ui": {"widget": "toggle", "label": "Private"}
						},
						"auto_init": {
							"type": "boolean",
							"default": false,
							"description": "Whether to initialize with a README",
							"x-ui": {"widget": "toggle", "label": "Initialize with README"}
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "github",
				AuthType:        "oauth2",
				OAuthProvider:   "github",
				OAuthScopes:     []string{"repo"},
				AuthOptionGroup: "github_auth",
			},
			{
				Service:         "github_pat",
				AuthType:        "api_key",
				InstructionsURL: "https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens",
				AuthOptionGroup: "github_auth",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:               "tpl_github_create_issue_all",
				ActionType:       "github.create_issue",
				Name:             "Create issues (all fields open)",
				Description:      "Agent can create issues in any repo with any title and body.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","title":"*","body":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_create_issue_org",
				ActionType:       "github.create_issue",
				Name:             "Create issues in your org",
				Description:      "Restricts the owner to your organization pattern. Agent can choose the repo, title, and body.",
				Parameters:       json.RawMessage(`{"owner":{"$pattern":"your-org-*"},"repo":"*","title":"*","body":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_merge_pr",
				ActionType:       "github.merge_pr",
				Name:             "Merge pull requests",
				Description:      "Agent can merge any PR. Owner, repo, and PR number are agent-controlled.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","pull_number":"*","merge_method":"squash"}`),
				StandingApproval: neverExpire,
			},
			// --- PR lifecycle templates ---
			{
				ID:               "tpl_github_create_pr",
				ActionType:       "github.create_pr",
				Name:             "Create pull requests",
				Description:      "Agent can create PRs in any repo.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","title":"*","body":"*","head":"*","base":"*","draft":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_create_pr_org",
				ActionType:       "github.create_pr",
				Name:             "Create PRs in your org",
				Description:      "Agent can create PRs only in repos owned by your organization.",
				Parameters:       json.RawMessage(`{"owner":{"$pattern":"your-org-*"},"repo":"*","title":"*","body":"*","head":"*","base":"*","draft":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_add_reviewer",
				ActionType:       "github.add_reviewer",
				Name:             "Add reviewers to PRs",
				Description:      "Agent can request reviewers on any PR.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","pull_number":"*","reviewers":"*"}`),
				StandingApproval: neverExpire,
			},
			// --- Release management templates ---
			{
				ID:               "tpl_github_create_release",
				ActionType:       "github.create_release",
				Name:             "Create releases",
				Description:      "Agent can create releases in any repo.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","tag_name":"*","name":"*","body":"*","draft":"*","prerelease":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_create_release_draft",
				ActionType:       "github.create_release",
				Name:             "Create draft releases only",
				Description:      "Agent can create draft releases — they won't be published until manually reviewed.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","tag_name":"*","name":"*","body":"*","draft":true,"prerelease":"*"}`),
				StandingApproval: neverExpire,
			},
			// --- Issue lifecycle templates ---
			{
				ID:               "tpl_github_close_issue",
				ActionType:       "github.close_issue",
				Name:             "Close issues",
				Description:      "Agent can close issues in any repo with an optional comment.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","issue_number":"*","state_reason":"*","comment":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_close_issue_completed",
				ActionType:       "github.close_issue",
				Name:             "Close issues as completed",
				Description:      "Agent can close issues as completed (not as not_planned). Useful for bots that resolve issues.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","issue_number":"*","state_reason":"completed","comment":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_add_label",
				ActionType:       "github.add_label",
				Name:             "Add labels",
				Description:      "Agent can add labels to any issue or PR.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","issue_number":"*","labels":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_add_comment",
				ActionType:       "github.add_comment",
				Name:             "Add comments",
				Description:      "Agent can comment on any issue or PR.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","issue_number":"*","body":"*"}`),
				StandingApproval: neverExpire,
			},
			// --- Branch management templates ---
			{
				ID:               "tpl_github_create_branch",
				ActionType:       "github.create_branch",
				Name:             "Create branches",
				Description:      "Agent can create branches in any repo.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","branch_name":"*","from_ref":"*"}`),
				StandingApproval: neverExpire,
			},
			// --- File contents templates ---
			{
				ID:               "tpl_github_get_file_contents",
				ActionType:       "github.get_file_contents",
				Name:             "Read files from any repo",
				Description:      "Agent can read any file from any repository.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","path":"*","ref":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_create_or_update_file",
				ActionType:       "github.create_or_update_file",
				Name:             "Create or update files",
				Description:      "Agent can create or update files in any repository.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","path":"*","message":"*","content":"*","branch":"*","sha":"*"}`),
				StandingApproval: neverExpire,
			},
			// --- Repo discovery templates ---
			{
				ID:               "tpl_github_list_repos",
				ActionType:       "github.list_repos",
				Name:             "List repositories",
				Description:      "Agent can list repositories for the authenticated user or any organization.",
				Parameters:       json.RawMessage(`{"org":"*","type":"*","sort":"*","per_page":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_get_repo",
				ActionType:       "github.get_repo",
				Name:             "Get repository metadata",
				Description:      "Agent can fetch metadata for any repository.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_list_pull_requests",
				ActionType:       "github.list_pull_requests",
				Name:             "List pull requests",
				Description:      "Agent can list pull requests from any repository.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","state":"*","base":"*","head":"*","sort":"*","per_page":"*"}`),
				StandingApproval: neverExpire,
			},
			// --- CI/CD templates ---
			{
				ID:               "tpl_github_trigger_workflow",
				ActionType:       "github.trigger_workflow",
				Name:             "Trigger any workflow",
				Description:      "Agent can trigger workflow dispatch events in any repository.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","workflow_id":"*","ref":"*","inputs":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_list_workflow_runs",
				ActionType:       "github.list_workflow_runs",
				Name:             "List workflow runs",
				Description:      "Agent can list workflow run history for any repository.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","workflow_id":"*","status":"*","branch":"*","per_page":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_create_webhook",
				ActionType:       "github.create_webhook",
				Name:             "Create webhooks",
				Description:      "Agent can create repository webhooks.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","url":"*","events":"*","content_type":"*","secret":"*","active":"*"}`),
				StandingApproval: neverExpire,
			},
			// --- Search templates ---
			{
				ID:               "tpl_github_search_code",
				ActionType:       "github.search_code",
				Name:             "Search code",
				Description:      "Agent can search code across all accessible repositories.",
				Parameters:       json.RawMessage(`{"q":"*","per_page":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_search_issues",
				ActionType:       "github.search_issues",
				Name:             "Search issues and PRs",
				Description:      "Agent can search issues and pull requests across all accessible repositories.",
				Parameters:       json.RawMessage(`{"q":"*","sort":"*","order":"*","per_page":"*"}`),
				StandingApproval: neverExpire,
			},
			// --- Repo creation templates ---
			{
				ID:               "tpl_github_create_repo",
				ActionType:       "github.create_repo",
				Name:             "Create repositories",
				Description:      "Agent can create repositories for the authenticated user or any organization.",
				Parameters:       json.RawMessage(`{"name":"*","org":"*","description":"*","private":"*","auto_init":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_create_repo_private",
				ActionType:       "github.create_repo",
				Name:             "Create private repositories only",
				Description:      "Agent can create repositories but they must be private.",
				Parameters:       json.RawMessage(`{"name":"*","org":"*","description":"*","private":true,"auto_init":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_merge_pr_org",
				ActionType:       "github.merge_pr",
				Name:             "Merge PRs in your org",
				Description:      "Merge pull requests only when the repo owner matches your organization pattern.",
				Parameters:       json.RawMessage(`{"owner":{"$pattern":"your-org-*"},"repo":"*","pull_number":"*","merge_method":"squash"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_add_comment_org",
				ActionType:       "github.add_comment",
				Name:             "Comment in org repos only",
				Description:      "Add issue/PR comments only in repositories owned by your organization.",
				Parameters:       json.RawMessage(`{"owner":{"$pattern":"your-org-*"},"repo":"*","issue_number":"*","body":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_get_file_contents_org",
				ActionType:       "github.get_file_contents",
				Name:             "Read files from org repos only",
				Description:      "Read repository files only when owner matches your organization pattern.",
				Parameters:       json.RawMessage(`{"owner":{"$pattern":"your-org-*"},"repo":"*","path":"*","ref":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_trigger_workflow_repo",
				ActionType:       "github.trigger_workflow",
				Name:             "Trigger workflows in one repo",
				Description:      "Locks owner and repository — agent can choose workflow, ref, and inputs.",
				Parameters:       json.RawMessage(`{"owner":"your-org","repo":"your-repo","workflow_id":"*","ref":"*","inputs":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_create_branch_org",
				ActionType:       "github.create_branch",
				Name:             "Create branches in org repos only",
				Description:      "Create branches only in repositories owned by your organization.",
				Parameters:       json.RawMessage(`{"owner":{"$pattern":"your-org-*"},"repo":"*","branch_name":"*","from_ref":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_delete_branch",
				ActionType:       "github.delete_branch",
				Name:             "Delete branches",
				Description:      "Agent can delete branch refs in any repository.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","branch_name":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_delete_file",
				ActionType:       "github.delete_file",
				Name:             "Delete files",
				Description:      "Agent can delete files when commit message and file SHA are provided.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","path":"*","message":"*","sha":"*","branch":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_remove_label",
				ActionType:       "github.remove_label",
				Name:             "Remove labels",
				Description:      "Agent can remove a label from an issue or pull request.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","issue_number":"*","name":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_close_pr",
				ActionType:       "github.close_pr",
				Name:             "Close pull requests",
				Description:      "Agent can close open pull requests without merging.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","pull_number":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_update_issue",
				ActionType:       "github.update_issue",
				Name:             "Update issues",
				Description:      "Agent can update issue title, body, state, or assignees.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","issue_number":"*","title":"*","body":"*","state":"*","assignees":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_update_pr",
				ActionType:       "github.update_pr",
				Name:             "Update pull requests",
				Description:      "Agent can update PR title, body, base branch, or state.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","pull_number":"*","title":"*","body":"*","base":"*","state":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_get_issue",
				ActionType:       "github.get_issue",
				Name:             "Get issue details",
				Description:      "Agent can fetch a single issue by number.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","issue_number":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_get_pr",
				ActionType:       "github.get_pr",
				Name:             "Get pull request details",
				Description:      "Agent can fetch a single pull request by number.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","pull_number":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_list_issues",
				ActionType:       "github.list_issues",
				Name:             "List issues",
				Description:      "Agent can list repository issues (pull requests excluded by default).",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","state":"*","labels":"*","sort":"*","direction":"*","since":"*","include_pull_requests":"*","per_page":"*","page":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_list_commits",
				ActionType:       "github.list_commits",
				Name:             "List commits",
				Description:      "Agent can list commits for a repository.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","sha":"*","path":"*","author":"*","per_page":"*","page":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_remove_reviewer",
				ActionType:       "github.remove_reviewer",
				Name:             "Remove requested reviewers",
				Description:      "Agent can remove users from a PR's requested reviewers list.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","pull_number":"*","reviewers":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_github_delete_webhook",
				ActionType:       "github.delete_webhook",
				Name:             "Delete webhooks",
				Description:      "Agent can delete repository webhooks by hook ID.",
				Parameters:       json.RawMessage(`{"owner":"*","repo":"*","hook_id":"*"}`),
				StandingApproval: neverExpire,
			},
		},
	}
}
