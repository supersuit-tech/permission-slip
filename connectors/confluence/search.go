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

// maxSearchLimit caps the limit parameter to prevent unbounded result sets.
const maxSearchLimit = 250

// searchAction implements connectors.Action for confluence.search.
// It searches pages using CQL via GET /wiki/api/v2/search.
type searchAction struct {
	conn *ConfluenceConnector
}

type searchParams struct {
	CQL   string `json:"cql"`
	Limit int    `json:"limit"`
}

func (p *searchParams) validate() error {
	p.CQL = strings.TrimSpace(p.CQL)
	if p.CQL == "" {
		return &connectors.ValidationError{Message: "missing required parameter: cql"}
	}
	if p.Limit > maxSearchLimit {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit cannot exceed %d", maxSearchLimit)}
	}
	return nil
}

func (a *searchAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	if params.Limit <= 0 {
		params.Limit = 25
	}

	path := fmt.Sprintf("/search?cql=%s&limit=%d", url.QueryEscape(params.CQL), params.Limit)

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
