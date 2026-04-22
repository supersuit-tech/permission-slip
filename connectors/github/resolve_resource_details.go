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
	switch actionType {
	case "github.trigger_workflow":
		return c.resolveWorkflow(ctx, creds, params)
	case "github.delete_webhook":
		return c.resolveWebhook(ctx, creds, params)
	default:
		return nil, nil
	}
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
		Name string `json:"name"`
	}
	if err := c.do(ctx, creds, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}
	if strings.TrimSpace(resp.Name) == "" {
		return nil, nil
	}
	return map[string]any{"workflow_name": resp.Name}, nil
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
