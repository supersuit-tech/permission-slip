package walmart

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getProductAction implements connectors.Action for walmart.get_product.
// It fetches product details via GET /items/{id}.
type getProductAction struct {
	conn *WalmartConnector
}

type getProductParams struct {
	ItemID string `json:"item_id"`
}

func (p *getProductParams) validate() error {
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_id (e.g. \"12345678\")"}
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
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/items/"+params.ItemID+"?format=json", &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(json.RawMessage(resp))
}
