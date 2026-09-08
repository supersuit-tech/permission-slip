package github

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

var _ connectors.ResourceDetailResolver = (*GitHubConnector)(nil)

// ResolveResourceDetails fetches human-readable metadata for resources
// referenced by opaque IDs in GitHub action parameters. Errors are non-fatal —
// the caller stores the approval without details on failure.
func (c *GitHubConnector) ResolveResourceDetails(ctx context.Context, actionType string, params json.RawMessage, creds connectors.Credentials) (map[string]any, error) {
	var details map[string]any
	var err error
	switch actionType {
	case "github.trigger_workflow":
		details, err = c.resolveWorkflow(ctx, creds, params)
	case "github.delete_webhook":
		details, err = c.resolveWebhook(ctx, creds, params)
	default:
		details = map[string]any{}
	}
	if err != nil {
		return nil, err
	}
	if details == nil {
		details = map[string]any{}
	}
	attachGitHubParamLinks(details, params)
	if len(details) == 0 {
		return nil, nil
	}
	return details, nil
}

func attachGitHubParamLinks(details map[string]any, params json.RawMessage) {
	var p struct {
		Owner       string          `json:"owner"`
		Repo        string          `json:"repo"`
		IssueNumber json.RawMessage `json:"issue_number"`
		PullNumber  json.RawMessage `json:"pull_number"`
		Path        string          `json:"path"`
		BranchName  string          `json:"branch_name"`
		WorkflowID  string          `json:"workflow_id"`
		HookID      json.RawMessage `json:"hook_id"`
	}
	if json.Unmarshal(params, &p) != nil {
		return
	}
	if p.Owner == "" || p.Repo == "" {
		return
	}
	repoURL := fmt.Sprintf("https://github.com/%s/%s", url.PathEscape(p.Owner), url.PathEscape(p.Repo))
	repoLabel := p.Owner + "/" + p.Repo
	connectors.AttachResources(details, connectors.ResourceRef{
		Param: "owner",
		ID:    p.Owner,
		Name:  p.Owner,
		URL:   "https://github.com/" + url.PathEscape(p.Owner),
	})
	connectors.AttachResources(details, connectors.ResourceRef{
		Param: "repo",
		ID:    p.Repo,
		Name:  repoLabel,
		URL:   repoURL,
	})
	if n := jsonFlexibleInt(p.IssueNumber); n > 0 {
		label := fmt.Sprintf("%s#%d", repoLabel, n)
		issueURL := fmt.Sprintf("%s/issues/%d", repoURL, n)
		connectors.AttachResources(details, connectors.ResourceRef{
			Param: "issue_number",
			ID:    fmt.Sprintf("%d", n),
			Name:  label,
			URL:   issueURL,
		})
	}
	if n := jsonFlexibleInt(p.PullNumber); n > 0 {
		label := fmt.Sprintf("%s#%d", repoLabel, n)
		prURL := fmt.Sprintf("%s/pull/%d", repoURL, n)
		connectors.AttachResources(details, connectors.ResourceRef{
			Param: "pull_number",
			ID:    fmt.Sprintf("%d", n),
			Name:  label,
			URL:   prURL,
		})
	}
	if p.Path != "" {
		fileURL := repoURL + "/blob/HEAD/" + strings.TrimPrefix(p.Path, "/")
		if p.BranchName != "" {
			fileURL = repoURL + "/blob/" + url.PathEscape(p.BranchName) + "/" + strings.TrimPrefix(p.Path, "/")
		}
		connectors.AttachResources(details, connectors.ResourceRef{
			Param: "path",
			ID:    p.Path,
			Name:  p.Path,
			URL:   fileURL,
		})
	}
}

func jsonFlexibleInt(raw json.RawMessage) int {
	if len(raw) == 0 || string(raw) == "null" {
		return 0
	}
	var n int
	if json.Unmarshal(raw, &n) == nil {
		return n
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		var parsed int
		_, _ = fmt.Sscanf(s, "%d", &parsed)
		return parsed
	}
	return 0
}

func (c *GitHubConnector) resolveWorkflow(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		Owner      string `json:"owner"`
		Repo       string `json:"repo"`
		WorkflowID string `json:"workflow_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("parse params: %w", err)
	}
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return nil, err
	}
	if p.WorkflowID == "" {
		return nil, &connectors.ValidationError{Message: "missing required parameter: workflow_id"}
	}

	path := fmt.Sprintf("/repos/%s/%s/actions/workflows/%s",
		url.PathEscape(p.Owner), url.PathEscape(p.Repo), url.PathEscape(p.WorkflowID))

	var resp struct {
		Name    string `json:"name"`
		HTMLURL string `json:"html_url"`
	}
	if err := c.do(ctx, creds, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.Name) == "" {
		return nil, nil
	}
	details := map[string]any{"workflow_name": resp.Name}
	return connectors.AttachResources(details, connectors.ResourceRef{
		Param: "workflow_id",
		ID:    p.WorkflowID,
		Name:  resp.Name,
		URL:   strings.TrimSpace(resp.HTMLURL),
	}), nil
}

func (c *GitHubConnector) resolveWebhook(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p struct {
		Owner  string `json:"owner"`
		Repo   string `json:"repo"`
		HookID int    `json:"hook_id"`
	}
	if err := json.Unmarshal(params, &p); err != nil {
		return nil, fmt.Errorf("parse params: %w", err)
	}
	if err := requireOwnerRepo(p.Owner, p.Repo); err != nil {
		return nil, err
	}
	if err := requirePositiveInt(p.HookID, "hook_id"); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/repos/%s/%s/hooks/%d",
		url.PathEscape(p.Owner), url.PathEscape(p.Repo), p.HookID)

	var resp struct {
		Config struct {
			URL string `json:"url"`
		} `json:"config"`
		Events []string `json:"events"`
	}
	if err := c.do(ctx, creds, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	out := map[string]any{}
	if u := strings.TrimSpace(resp.Config.URL); u != "" {
		out["webhook_url"] = u
	}
	if n := len(resp.Events); n > 0 {
		if n <= 3 {
			out["webhook_events"] = strings.Join(resp.Events, ", ")
		} else {
			out["webhook_events"] = fmt.Sprintf("%s, +%d more", strings.Join(resp.Events[:3], ", "), n-3)
		}
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
