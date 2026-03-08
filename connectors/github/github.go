// Package github implements the GitHub connector for the Permission Slip
// connector execution layer. It uses the GitHub REST API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package github

import (
	_ "embed"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://api.github.com"
	defaultTimeout = 30 * time.Second
)

// GitHubConnector owns the shared HTTP client and base URL used by all
// GitHub actions. Actions hold a pointer back to the connector to access
// these shared resources.
type GitHubConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a GitHubConnector with sensible defaults (30s timeout,
// https://api.github.com base URL).
func New() *GitHubConnector {
	return &GitHubConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a GitHubConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *GitHubConnector {
	return &GitHubConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "github", matching the connectors.id in the database.
func (c *GitHubConnector) ID() string { return "github" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
//go:embed logo.svg
var logoSVG string

func (c *GitHubConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "github",
		Name:        "GitHub",
		Description: "GitHub integration for repository management",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "github.create_issue",
				Name:        "Create Issue",
				Description: "Create a new issue in a repository",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "title"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"title": {
							"type": "string",
							"description": "Issue title"
						},
						"body": {
							"type": "string",
							"description": "Issue body (Markdown supported)"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.merge_pr",
				Name:        "Merge Pull Request",
				Description: "Merge an open pull request",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "pull_number"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"pull_number": {
							"type": "integer",
							"description": "Pull request number"
						},
						"merge_method": {
							"type": "string",
							"enum": ["merge", "squash", "rebase"],
							"default": "merge",
							"description": "Merge strategy to use"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.create_pr",
				Name:        "Create Pull Request",
				Description: "Create a pull request from a branch",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "title", "head", "base"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"title": {
							"type": "string",
							"description": "Pull request title"
						},
						"body": {
							"type": "string",
							"description": "Pull request body (Markdown supported)"
						},
						"head": {
							"type": "string",
							"description": "Branch containing the changes"
						},
						"base": {
							"type": "string",
							"description": "Branch to merge into"
						},
						"draft": {
							"type": "boolean",
							"default": false,
							"description": "Whether to create the PR as a draft"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.add_reviewer",
				Name:        "Add Reviewer",
				Description: "Request reviews on a pull request",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "pull_number", "reviewers"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"pull_number": {
							"type": "integer",
							"description": "Pull request number"
						},
						"reviewers": {
							"type": "array",
							"items": {"type": "string"},
							"description": "GitHub usernames to request reviews from"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.create_release",
				Name:        "Create Release",
				Description: "Create a tagged release in a repository",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "tag_name"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"tag_name": {
							"type": "string",
							"description": "The name of the tag for this release"
						},
						"name": {
							"type": "string",
							"description": "The name of the release"
						},
						"body": {
							"type": "string",
							"description": "Release notes (Markdown supported)"
						},
						"draft": {
							"type": "boolean",
							"default": false,
							"description": "Whether to create as a draft release"
						},
						"prerelease": {
							"type": "boolean",
							"default": false,
							"description": "Whether to mark as a pre-release"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.close_issue",
				Name:        "Close Issue",
				Description: "Close an issue with an optional comment",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "issue_number"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"issue_number": {
							"type": "integer",
							"description": "Issue number to close"
						},
						"state_reason": {
							"type": "string",
							"enum": ["completed", "not_planned"],
							"default": "completed",
							"description": "Reason for closing the issue"
						},
						"comment": {
							"type": "string",
							"description": "Optional comment to add before closing"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.add_label",
				Name:        "Add Label",
				Description: "Add labels to an issue or pull request",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "issue_number", "labels"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"issue_number": {
							"type": "integer",
							"description": "Issue or pull request number"
						},
						"labels": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Labels to add"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.add_comment",
				Name:        "Add Comment",
				Description: "Add a comment to an issue or pull request",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "issue_number", "body"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"issue_number": {
							"type": "integer",
							"description": "Issue or pull request number"
						},
						"body": {
							"type": "string",
							"description": "Comment body (Markdown supported)"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.create_branch",
				Name:        "Create Branch",
				Description: "Create a new branch from a ref",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "branch_name", "from_ref"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"branch_name": {
							"type": "string",
							"description": "Name for the new branch"
						},
						"from_ref": {
							"type": "string",
							"description": "Branch or ref to create from (e.g. \"main\", \"develop\", or \"tags/v1.0\")"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.get_file_contents",
				Name:        "Get File Contents",
				Description: "Read file contents from a repository",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "path"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"path": {
							"type": "string",
							"description": "File path within the repository"
						},
						"ref": {
							"type": "string",
							"description": "Branch, tag, or commit SHA to read from (defaults to default branch)"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.create_or_update_file",
				Name:        "Create or Update File",
				Description: "Create or update a file in a repository via the Contents API",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "path", "message", "content"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"path": {
							"type": "string",
							"description": "File path within the repository"
						},
						"message": {
							"type": "string",
							"description": "Commit message"
						},
						"content": {
							"type": "string",
							"description": "New file content, base64-encoded"
						},
						"branch": {
							"type": "string",
							"description": "Branch to commit to (defaults to default branch)"
						},
						"sha": {
							"type": "string",
							"description": "Blob SHA of the file being replaced (required when updating an existing file)"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.list_repos",
				Name:        "List Repositories",
				Description: "List repositories for the authenticated user or an organization",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"org": {
							"type": "string",
							"description": "Organization login to list repos for (omit to list the authenticated user's repos)"
						},
						"type": {
							"type": "string",
							"enum": ["all", "public", "private", "forks", "sources", "member"],
							"description": "Repository type filter"
						},
						"sort": {
							"type": "string",
							"enum": ["created", "updated", "pushed", "full_name"],
							"description": "Sort field"
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Number of results per page (max 100)"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.get_repo",
				Name:        "Get Repository",
				Description: "Get repository metadata",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.list_pull_requests",
				Name:        "List Pull Requests",
				Description: "List pull requests for a repository with optional filtering",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"state": {
							"type": "string",
							"enum": ["open", "closed", "all"],
							"default": "open",
							"description": "State filter"
						},
						"base": {
							"type": "string",
							"description": "Filter by base branch name"
						},
						"head": {
							"type": "string",
							"description": "Filter by head branch name (user:branch format)"
						},
						"sort": {
							"type": "string",
							"enum": ["created", "updated", "popularity", "long-running"],
							"description": "Sort field"
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Number of results per page (max 100)"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.trigger_workflow",
				Name:        "Trigger Workflow",
				Description: "Trigger a GitHub Actions workflow dispatch event",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "workflow_id", "ref"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"workflow_id": {
							"type": "string",
							"description": "Workflow file name (e.g. \"deploy.yml\") or numeric workflow ID"
						},
						"ref": {
							"type": "string",
							"description": "Branch or tag to run the workflow on"
						},
						"inputs": {
							"type": "object",
							"description": "Input key-value pairs defined by the workflow's on.workflow_dispatch.inputs",
							"additionalProperties": true
						}
					}
				}`)),
			},
			{
				ActionType:  "github.list_workflow_runs",
				Name:        "List Workflow Runs",
				Description: "List workflow runs for a repository with optional status filtering",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"workflow_id": {
							"type": "string",
							"description": "Workflow file name or ID to filter runs (omit for all workflows)"
						},
						"status": {
							"type": "string",
							"enum": ["completed", "action_required", "cancelled", "failure", "neutral", "skipped", "stale", "success", "timed_out", "in_progress", "queued", "requested", "waiting", "pending"],
							"description": "Filter by run status"
						},
						"branch": {
							"type": "string",
							"description": "Filter by branch name"
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Number of results per page (max 100)"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.create_webhook",
				Name:        "Create Webhook",
				Description: "Create a repository webhook",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["owner", "repo", "url", "events"],
					"properties": {
						"owner": {
							"type": "string",
							"description": "Repository owner (user or organization)"
						},
						"repo": {
							"type": "string",
							"description": "Repository name"
						},
						"url": {
							"type": "string",
							"description": "Payload URL for the webhook"
						},
						"events": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Events that trigger the webhook (e.g. [\"push\", \"pull_request\"])"
						},
						"content_type": {
							"type": "string",
							"enum": ["json", "form"],
							"default": "json",
							"description": "Payload content type"
						},
						"secret": {
							"type": "string",
							"description": "Secret used to sign webhook payloads"
						},
						"active": {
							"type": "boolean",
							"default": true,
							"description": "Whether the webhook is active"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.search_code",
				Name:        "Search Code",
				Description: "Search code across repositories",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["q"],
					"properties": {
						"q": {
							"type": "string",
							"description": "Search query (supports GitHub code search qualifiers)"
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Number of results per page (max 100)"
						}
					}
				}`)),
			},
			{
				ActionType:  "github.search_issues",
				Name:        "Search Issues",
				Description: "Search issues and pull requests across repositories",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["q"],
					"properties": {
						"q": {
							"type": "string",
							"description": "Search query (supports GitHub issue search qualifiers)"
						},
						"sort": {
							"type": "string",
							"enum": ["comments", "reactions", "reactions-+1", "reactions--1", "reactions-smile", "reactions-thinking_face", "reactions-heart", "reactions-tada", "interactions", "created", "updated"],
							"description": "Sort field"
						},
						"order": {
							"type": "string",
							"enum": ["asc", "desc"],
							"description": "Sort order"
						},
						"per_page": {
							"type": "integer",
							"default": 30,
							"description": "Number of results per page (max 100)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "github",
				AuthType:      "oauth2",
				OAuthProvider: "github",
				OAuthScopes:   []string{"repo"},
			},
			{
				Service:         "github_pat",
				AuthType:        "api_key",
				InstructionsURL: "https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/managing-your-personal-access-tokens",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_github_create_issue_all",
				ActionType:  "github.create_issue",
				Name:        "Create issues (all fields open)",
				Description: "Agent can create issues in any repo with any title and body.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","title":"*","body":"*"}`),
			},
			{
				ID:          "tpl_github_create_issue_org",
				ActionType:  "github.create_issue",
				Name:        "Create issues in your org",
				Description: "Restricts the owner to your organization pattern. Agent can choose the repo, title, and body.",
				Parameters:  json.RawMessage(`{"owner":{"$pattern":"your-org-*"},"repo":"*","title":"*","body":"*"}`),
			},
			{
				ID:          "tpl_github_merge_pr",
				ActionType:  "github.merge_pr",
				Name:        "Merge pull requests",
				Description: "Agent can merge any PR. Owner, repo, and PR number are agent-controlled.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","pull_number":"*","merge_method":"squash"}`),
			},
			// --- PR lifecycle templates ---
			{
				ID:          "tpl_github_create_pr",
				ActionType:  "github.create_pr",
				Name:        "Create pull requests",
				Description: "Agent can create PRs in any repo.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","title":"*","body":"*","head":"*","base":"*","draft":"*"}`),
			},
			{
				ID:          "tpl_github_create_pr_org",
				ActionType:  "github.create_pr",
				Name:        "Create PRs in your org",
				Description: "Agent can create PRs only in repos owned by your organization.",
				Parameters:  json.RawMessage(`{"owner":{"$pattern":"your-org-*"},"repo":"*","title":"*","body":"*","head":"*","base":"*","draft":"*"}`),
			},
			{
				ID:          "tpl_github_add_reviewer",
				ActionType:  "github.add_reviewer",
				Name:        "Add reviewers to PRs",
				Description: "Agent can request reviewers on any PR.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","pull_number":"*","reviewers":"*"}`),
			},
			// --- Release management templates ---
			{
				ID:          "tpl_github_create_release",
				ActionType:  "github.create_release",
				Name:        "Create releases",
				Description: "Agent can create releases in any repo.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","tag_name":"*","name":"*","body":"*","draft":"*","prerelease":"*"}`),
			},
			{
				ID:          "tpl_github_create_release_draft",
				ActionType:  "github.create_release",
				Name:        "Create draft releases only",
				Description: "Agent can create draft releases — they won't be published until manually reviewed.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","tag_name":"*","name":"*","body":"*","draft":true,"prerelease":"*"}`),
			},
			// --- Issue lifecycle templates ---
			{
				ID:          "tpl_github_close_issue",
				ActionType:  "github.close_issue",
				Name:        "Close issues",
				Description: "Agent can close issues in any repo with an optional comment.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","issue_number":"*","state_reason":"*","comment":"*"}`),
			},
			{
				ID:          "tpl_github_close_issue_completed",
				ActionType:  "github.close_issue",
				Name:        "Close issues as completed",
				Description: "Agent can close issues as completed (not as not_planned). Useful for bots that resolve issues.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","issue_number":"*","state_reason":"completed","comment":"*"}`),
			},
			{
				ID:          "tpl_github_add_label",
				ActionType:  "github.add_label",
				Name:        "Add labels",
				Description: "Agent can add labels to any issue or PR.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","issue_number":"*","labels":"*"}`),
			},
			{
				ID:          "tpl_github_add_comment",
				ActionType:  "github.add_comment",
				Name:        "Add comments",
				Description: "Agent can comment on any issue or PR.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","issue_number":"*","body":"*"}`),
			},
			// --- Branch management templates ---
			{
				ID:          "tpl_github_create_branch",
				ActionType:  "github.create_branch",
				Name:        "Create branches",
				Description: "Agent can create branches in any repo.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","branch_name":"*","from_ref":"*"}`),
			},
			// --- File contents templates ---
			{
				ID:          "tpl_github_get_file_contents",
				ActionType:  "github.get_file_contents",
				Name:        "Read files from any repo",
				Description: "Agent can read any file from any repository.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","path":"*","ref":"*"}`),
			},
			{
				ID:          "tpl_github_create_or_update_file",
				ActionType:  "github.create_or_update_file",
				Name:        "Create or update files",
				Description: "Agent can create or update files in any repository.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","path":"*","message":"*","content":"*","branch":"*","sha":"*"}`),
			},
			// --- Repo discovery templates ---
			{
				ID:          "tpl_github_list_repos",
				ActionType:  "github.list_repos",
				Name:        "List repositories",
				Description: "Agent can list repositories for the authenticated user or any organization.",
				Parameters:  json.RawMessage(`{"org":"*","type":"*","sort":"*","per_page":"*"}`),
			},
			{
				ID:          "tpl_github_get_repo",
				ActionType:  "github.get_repo",
				Name:        "Get repository metadata",
				Description: "Agent can fetch metadata for any repository.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*"}`),
			},
			{
				ID:          "tpl_github_list_pull_requests",
				ActionType:  "github.list_pull_requests",
				Name:        "List pull requests",
				Description: "Agent can list pull requests from any repository.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","state":"*","base":"*","head":"*","sort":"*","per_page":"*"}`),
			},
			// --- CI/CD templates ---
			{
				ID:          "tpl_github_trigger_workflow",
				ActionType:  "github.trigger_workflow",
				Name:        "Trigger any workflow",
				Description: "Agent can trigger workflow dispatch events in any repository.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","workflow_id":"*","ref":"*","inputs":"*"}`),
			},
			{
				ID:          "tpl_github_list_workflow_runs",
				ActionType:  "github.list_workflow_runs",
				Name:        "List workflow runs",
				Description: "Agent can list workflow run history for any repository.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","workflow_id":"*","status":"*","branch":"*","per_page":"*"}`),
			},
			{
				ID:          "tpl_github_create_webhook",
				ActionType:  "github.create_webhook",
				Name:        "Create webhooks",
				Description: "Agent can create repository webhooks.",
				Parameters:  json.RawMessage(`{"owner":"*","repo":"*","url":"*","events":"*","content_type":"*","secret":"*","active":"*"}`),
			},
			// --- Search templates ---
			{
				ID:          "tpl_github_search_code",
				ActionType:  "github.search_code",
				Name:        "Search code",
				Description: "Agent can search code across all accessible repositories.",
				Parameters:  json.RawMessage(`{"q":"*","per_page":"*"}`),
			},
			{
				ID:          "tpl_github_search_issues",
				ActionType:  "github.search_issues",
				Name:        "Search issues and PRs",
				Description: "Agent can search issues and pull requests across all accessible repositories.",
				Parameters:  json.RawMessage(`{"q":"*","sort":"*","order":"*","per_page":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *GitHubConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"github.create_issue":   &createIssueAction{conn: c},
		"github.merge_pr":       &mergePRAction{conn: c},
		"github.create_pr":      &createPRAction{conn: c},
		"github.add_reviewer":   &addReviewerAction{conn: c},
		"github.create_release": &createReleaseAction{conn: c},
		"github.close_issue":    &closeIssueAction{conn: c},
		"github.add_label":      &addLabelAction{conn: c},
		"github.add_comment":    &addCommentAction{conn: c},
		"github.create_branch":        &createBranchAction{conn: c},
		"github.get_file_contents":    &getFileContentsAction{conn: c},
		"github.create_or_update_file": &createOrUpdateFileAction{conn: c},
		"github.list_repos":            &listReposAction{conn: c},
		"github.get_repo":              &getRepoAction{conn: c},
		"github.list_pull_requests":    &listPullRequestsAction{conn: c},
		"github.trigger_workflow":      &triggerWorkflowAction{conn: c},
		"github.list_workflow_runs":    &listWorkflowRunsAction{conn: c},
		"github.create_webhook":        &createWebhookAction{conn: c},
		"github.search_code":           &searchCodeAction{conn: c},
		"github.search_issues":         &searchIssuesAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain either a
// non-empty access_token (from OAuth) or a non-empty api_key (PAT). OAuth
// tokens take precedence when both are present.
func (c *GitHubConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	if token, ok := creds.Get("access_token"); ok && token != "" {
		return nil
	}
	if key, ok := creds.Get("api_key"); ok && key != "" {
		return nil
	}
	return &connectors.ValidationError{Message: "missing required credential: access_token or api_key"}
}

// do is the shared request lifecycle for all GitHub actions. It marshals
// reqBody as JSON, sends the request with auth headers, checks the response
// status, and unmarshals the response into respBody. Either reqBody or
// respBody may be nil (e.g., DELETE with no body, or a request where the
// caller doesn't need the response).
func (c *GitHubConnector) do(ctx context.Context, creds connectors.Credentials, method, path string, reqBody, respBody interface{}) error {
	var body io.Reader
	if reqBody != nil {
		payload, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		body = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	if reqBody != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	token := ""
	if t, ok := creds.Get("access_token"); ok && t != "" {
		token = t
	} else if k, ok := creds.Get("api_key"); ok && k != "" {
		token = k
	}
	if token == "" {
		return &connectors.ValidationError{Message: "access_token or api_key credential is missing or empty"}
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("GitHub API request timed out: %v", err)}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("GitHub API request failed: %v", err)}
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := checkResponse(resp.StatusCode, resp.Header, respBytes); err != nil {
		return err
	}

	if respBody != nil {
		if err := json.Unmarshal(respBytes, respBody); err != nil {
			return &connectors.ExternalError{Message: fmt.Sprintf("parsing GitHub response: %v", err)}
		}
	}
	return nil
}

