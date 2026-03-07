package notion

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// appendBlocksAction implements connectors.Action for notion.append_blocks.
// It appends block objects to a page via PATCH /v1/blocks/{page_id}/children.
type appendBlocksAction struct {
	conn *NotionConnector
}

// appendBlocksParams is the user-facing parameter schema.
type appendBlocksParams struct {
	PageID   string          `json:"page_id"`
	Children json.RawMessage `json:"children,omitempty"`
	Text     string          `json:"text,omitempty"`
}

func (p *appendBlocksParams) validate() error {
	if err := validateNotionID(p.PageID, "page_id"); err != nil {
		return err
	}
	if len(p.Children) == 0 && p.Text == "" {
		return &connectors.ValidationError{Message: "at least one of children or text must be provided"}
	}
	return nil
}

// Execute appends blocks to a Notion page and returns the appended block metadata.
func (a *appendBlocksAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params appendBlocksParams
	if err := parseParams(req.Parameters, &params); err != nil {
		return nil, err
	}

	body, err := buildAppendBlocksBody(params)
	if err != nil {
		return nil, err
	}

	var resp map[string]any
	if err := a.conn.do(ctx, http.MethodPatch, "/v1/blocks/"+params.PageID+"/children", req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}

// buildAppendBlocksBody constructs the Notion API request body. If children is
// provided, it is used directly. Otherwise, if only text is provided, it is
// auto-wrapped as a single paragraph block for convenience.
func buildAppendBlocksBody(params appendBlocksParams) (map[string]any, error) {
	// If explicit children are provided, use them directly.
	if len(params.Children) > 0 {
		var children []any
		if err := json.Unmarshal(params.Children, &children); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid children JSON (expected array of block objects): %v", err)}
		}
		return map[string]any{"children": children}, nil
	}

	// Auto-wrap plain text as a paragraph block.
	return map[string]any{
		"children": []map[string]any{
			{
				"object": "block",
				"type":   "paragraph",
				"paragraph": map[string]any{
					"rich_text": []map[string]any{
						{"type": "text", "text": map[string]string{"content": params.Text}},
					},
				},
			},
		},
	}, nil
}
