package walmart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// itemIDPattern validates that Walmart item IDs contain only digits.
// Walmart item IDs are numeric (e.g. "12345678"). Rejecting non-numeric
// input prevents path injection even before url.PathEscape.
var itemIDPattern = regexp.MustCompile(`^\d+$`)

// getProductAction implements connectors.Action for walmart.get_product.
// It fetches product details via GET /items/{id}.
type getProductAction struct {
	conn *WalmartConnector
}

// getProductParams maps the JSON parameters for walmart.get_product.
type getProductParams struct {
	ItemID string `json:"item_id"`
}

func (p *getProductParams) validate() error {
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_id (e.g. \"12345678\")"}
	}
	if !itemIDPattern.MatchString(p.ItemID) {
		return &connectors.ValidationError{Message: fmt.Sprintf("item_id must be numeric, got %q", p.ItemID)}
	}
	if len(p.ItemID) > 20 {
		return &connectors.ValidationError{Message: fmt.Sprintf("item_id exceeds 20 characters (got %d)", len(p.ItemID))}
	}
	return nil
}

func (a *getProductAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getProductParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/items/"+url.PathEscape(params.ItemID)+"?format=json", &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(json.RawMessage(resp))
}
