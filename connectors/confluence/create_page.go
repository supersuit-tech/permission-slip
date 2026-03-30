package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createPageAction implements connectors.Action for confluence.create_page.
// It creates a new page via POST /wiki/api/v2/pages.
type createPageAction struct {
	conn *ConfluenceConnector
}

type createPageParams struct {
	SpaceID  string `json:"space_id"`
	Title    string `json:"title"`
	Body     string `json:"body"`
	ParentID string `json:"parent_id"`
	Status   string `json:"status"`
}

func (p *createPageParams) validate() error {
	p.SpaceID = strings.TrimSpace(p.SpaceID)
	p.Title = strings.TrimSpace(p.Title)
	if p.SpaceID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: space_id"}
	}
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	if strings.TrimSpace(p.Body) == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	if p.Status != "" && p.Status != "current" && p.Status != "draft" {
		return &connectors.ValidationError{Message: "status must be \"current\" or \"draft\""}
	}
	return nil
}

func (a *createPageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	status := params.Status
	if status == "" {
		status = "current"
	}

	reqBody := map[string]interface{}{
		"spaceId": params.SpaceID,
		"status":  status,
		"title":   params.Title,
		"body": map[string]interface{}{
			"representation": "storage",
			"value":          params.Body,
		},
	}
	if params.ParentID != "" {
		reqBody["parentId"] = params.ParentID
	}

	var resp pageResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/pages", reqBody, &resp); err != nil {
		return nil, err
	}

	return resp.toResult(req.Credentials)
}
