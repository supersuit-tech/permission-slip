package walmart

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getTaxonomyAction implements connectors.Action for walmart.get_taxonomy.
// It retrieves the product category taxonomy via GET /taxonomy.
type getTaxonomyAction struct {
	conn *WalmartConnector
}

func (a *getTaxonomyAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var resp json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, "/taxonomy?format=json", &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(json.RawMessage(resp))
}
