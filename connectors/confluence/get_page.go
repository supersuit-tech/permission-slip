package confluence

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getPageAction implements connectors.Action for confluence.get_page.
// It retrieves a page via GET /wiki/api/v2/pages/{page_id}.
type getPageAction struct {
	conn *ConfluenceConnector
}

type getPageParams struct {
	PageID     string `json:"page_id"`
	BodyFormat string `json:"body_format"`
}

func (p *getPageParams) validate() error {
	p.PageID = strings.TrimSpace(p.PageID)
	if p.PageID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: page_id"}
	}
	validFormats := map[string]bool{"storage": true, "atlas_doc_format": true, "view": true, "": true}
	if !validFormats[p.BodyFormat] {
		return &connectors.ValidationError{Message: "body_format must be one of: storage, atlas_doc_format, view"}
	}
	return nil
}

func (a *getPageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getPageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	bodyFormat := params.BodyFormat
	if bodyFormat == "" {
		bodyFormat = "storage"
	}

	path := "/pages/" + url.PathEscape(params.PageID) + "?body-format=" + url.QueryEscape(bodyFormat)

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
