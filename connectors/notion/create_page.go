package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createPageAction implements connectors.Action for notion.create_page.
// It creates a new page or database entry via POST /v1/pages.
type createPageAction struct {
	conn *NotionConnector
}

// createPageParams is the user-facing parameter schema.
type createPageParams struct {
	ParentID   string          `json:"parent_id"`
	Title      string          `json:"title"`
	Properties json.RawMessage `json:"properties,omitempty"`
	Content    json.RawMessage `json:"content,omitempty"`
}

func (p *createPageParams) validate() error {
	if p.ParentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: parent_id"}
	}
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	return nil
}

// Execute creates a page in Notion and returns the created page metadata.
func (a *createPageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createPageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body, err := buildCreatePageBody(params)
	if err != nil {
		return nil, err
	}

	var resp map[string]any
	if err := a.conn.do(ctx, http.MethodPost, "/v1/pages", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}

// buildCreatePageBody constructs the Notion API request body for creating a page.
func buildCreatePageBody(params createPageParams) (map[string]any, error) {
	body := map[string]any{
		"parent": map[string]string{
			"page_id": params.ParentID,
		},
		"properties": map[string]any{
			"title": []map[string]any{
				{"text": map[string]string{"content": params.Title}},
			},
		},
	}

	// If the caller provided explicit properties, use those instead of the
	// default title-only properties. This supports database entries where
	// properties follow the database schema.
	if len(params.Properties) > 0 {
		var props map[string]any
		if err := json.Unmarshal(params.Properties, &props); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid properties JSON: %v", err)}
		}
		body["properties"] = props
	}

	// Append child blocks if provided.
	if len(params.Content) > 0 {
		var children []any
		if err := json.Unmarshal(params.Content, &children); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid content JSON (expected array of block objects): %v", err)}
		}
		body["children"] = children
	}

	return body, nil
}
