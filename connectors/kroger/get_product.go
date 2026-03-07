package kroger

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getProductAction implements connectors.Action for kroger.get_product.
// It fetches product details via GET /v1/products/{id}.
type getProductAction struct {
	conn *KrogerConnector
}

type getProductParams struct {
	ProductID  string `json:"product_id"`
	LocationID string `json:"location_id"`
}

func (p *getProductParams) validate() error {
	if p.ProductID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: product_id"}
	}
	return nil
}

// Execute fetches details for a single Kroger product.
func (a *getProductAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getProductParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := "/products/" + url.PathEscape(params.ProductID)
	if params.LocationID != "" {
		path += "?filter.locationId=" + url.QueryEscape(params.LocationID)
	}

	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
