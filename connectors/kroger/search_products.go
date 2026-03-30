package kroger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// searchProductsAction implements connectors.Action for kroger.search_products.
// It searches products via GET /v1/products.
type searchProductsAction struct {
	conn *KrogerConnector
}

type searchProductsParams struct {
	Term       string `json:"term"`
	LocationID string `json:"location_id"`
	Limit      int    `json:"limit"`
	Start      int    `json:"start"`
}

func (p *searchProductsParams) validate() error {
	if p.Term == "" {
		return &connectors.ValidationError{Message: "missing required parameter: term"}
	}
	if p.Limit != 0 && (p.Limit < 1 || p.Limit > 50) {
		return &connectors.ValidationError{Message: fmt.Sprintf("limit must be between 1 and 50 (got %d)", p.Limit)}
	}
	if p.Start < 0 {
		return &connectors.ValidationError{Message: fmt.Sprintf("start must be non-negative (got %d)", p.Start)}
	}
	return nil
}

// Execute searches Kroger products by term.
func (a *searchProductsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchProductsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit == 0 {
		limit = 10
	}

	path := "/products?filter.term=" + url.QueryEscape(params.Term) +
		"&filter.limit=" + strconv.Itoa(limit)
	if params.LocationID != "" {
		path += "&filter.locationId=" + url.QueryEscape(params.LocationID)
	}
	if params.Start > 0 {
		path += "&filter.start=" + strconv.Itoa(params.Start)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
