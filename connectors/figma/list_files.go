package figma

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listFilesAction implements connectors.Action for figma.list_files.
// It lists files in a project via GET /projects/{project_id}/files.
type listFilesAction struct {
	conn *FigmaConnector
}

type listFilesParams struct {
	ProjectID string `json:"project_id"`
}

func (p *listFilesParams) validate() error {
	p.ProjectID = strings.TrimSpace(p.ProjectID)
	if p.ProjectID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: project_id"}
	}
	if strings.Contains(p.ProjectID, "/") || strings.Contains(p.ProjectID, "..") {
		return &connectors.ValidationError{Message: "project_id contains invalid characters"}
	}
	return nil
}

type listFilesResponse struct {
	Name  string      `json:"name"`
	Files []figmaFile `json:"files"`
}

type figmaFile struct {
	Key            string `json:"key"`
	Name           string `json:"name"`
	ThumbnailURL   string `json:"thumbnail_url"`
	LastModified   string `json:"last_modified"`
	EditorType     string `json:"editor_type"`
}

func (a *listFilesAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listFilesParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	var resp listFilesResponse
	path := fmt.Sprintf("/projects/%s/files", url.PathEscape(params.ProjectID))
	if err := a.conn.doGet(ctx, path, req.Credentials, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
