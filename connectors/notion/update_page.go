package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updatePageAction implements connectors.Action for notion.update_page.
// It updates properties on an existing page via PATCH /v1/pages/{page_id}.
type updatePageAction struct {
	conn *NotionConnector
}

// updatePageParams is the user-facing parameter schema.
type updatePageParams struct {
	PageID     string          `json:"page_id"`
	Properties json.RawMessage `json:"properties,omitempty"`
	Archived   *bool           `json:"archived,omitempty"`
}

func (p *updatePageParams) validate() error {
	if err := validateNotionID(p.PageID, "page_id"); err != nil {
		return err
	}
	if len(p.Properties) == 0 && p.Archived == nil {
		return &connectors.ValidationError{Message: "at least one of properties or archived must be provided"}
	}
	return nil
}

// Execute updates a Notion page and returns the updated page metadata.
func (a *updatePageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updatePageParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	body := make(map[string]any)
	if len(params.Properties) > 0 {
		var props map[string]any
		if err := json.Unmarshal(params.Properties, &props); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid properties: %v", err)}
		}
		body["properties"] = props
	}
	if params.Archived != nil {
		body["archived"] = *params.Archived
	}

	var resp map[string]any
	if err := a.conn.do(ctx, http.MethodPatch, "/v1/pages/"+params.PageID, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
