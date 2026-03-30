package jira

import (
	"context"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listProjectsAction implements connectors.Action for jira.list_projects.
// It lists projects via GET /rest/api/3/project and returns a clean
// response with only the fields most commonly needed.
type listProjectsAction struct {
	conn *JiraConnector
}

type jiraProject struct {
	ID          string `json:"id"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	ProjectType string `json:"projectTypeKey"`
	Style       string `json:"style"`
}

func (a *listProjectsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var rawProjects []jiraProject
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/project", nil, &rawProjects); err != nil {
		return nil, err
	}

	projects := make([]map[string]string, 0, len(rawProjects))
	for _, p := range rawProjects {
		projects = append(projects, map[string]string{
			"id":           p.ID,
			"key":          p.Key,
			"name":         p.Name,
			"project_type": p.ProjectType,
			"style":        p.Style,
		})
	}

	return connectors.JSONResult(map[string]interface{}{
		"projects":    projects,
		"total_count": len(projects),
	})
}
